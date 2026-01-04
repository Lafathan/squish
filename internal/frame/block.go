package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

type Block struct {
	BlockType uint8  // 0x00 EOS, 0x01 Default codec, 0x02 Block codec
	Codec     uint8  // only used if BlockType == 0x02
	USize     uint64 // uncompressed size
	CSize     uint64 // compressed size
	Checksum  uint64 // checksum value 4 bytes for uncompressed, 4 bytes for compressed
}

func (b *Block) Valid() error {
	if (b.BlockType != EOS) && (b.BlockType != DefaultCodec) && (b.BlockType != BlockCodec) {
		return errors.New("invalid block type found")
	}
	if b.USize > MaxBlockSize {
		return errors.New("invalid block size found")
	}
	return nil
}

func (b Block) String() string {
	s := fmt.Sprintf("BlockType:      %d\n", b.BlockType)
	s += fmt.Sprintf("Codec:          %d\n", b.Codec)
	s += fmt.Sprintf("USize:          %d\n", b.USize)
	s += fmt.Sprintf("CSize:          %d\n", b.CSize)
	s += fmt.Sprintf("Checksum:       %016x\n", b.Checksum)
	return s
}

func ReadBlock(fr *FrameReader) (Block, error) {
	var b Block
	var err error
	// get block type
	b.BlockType, err = fr.ReadByte()
	if err != nil {
		return b, fmt.Errorf("Error in reading block type: %v", err)
	}
	// return if EOS block
	if b.BlockType == EOS {
		return b, nil
	}
	// read the codec if there is a block specific one
	if b.BlockType == BlockCodec {
		b.Codec, err = fr.ReadByte()
		if err != nil {
			return b, fmt.Errorf("Error in reading block codec: %v", err)
		}
	}
	// read and assign the varint sizes
	b.USize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, fmt.Errorf("Error in reading block uncompressed size: %v", err)
	}
	b.CSize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, fmt.Errorf("Error in reading block compressed size: %v", err)
	}
	// read the checksum data according to the method
	byteLength := 0
	if fr.Header.ChecksumMode&CompressedChecksum != 0x00 {
		byteLength += crc32.Size
	}
	if fr.Header.ChecksumMode&UncompressedChecksum != 0x00 {
		byteLength += crc32.Size
	}
	if byteLength > 0 {
		cs, err := fr.ReadBytes(byteLength)
		if err != nil {
			return b, fmt.Errorf("Error in reading block checksum: %v", err)
		}
		for _, csbyte := range cs {
			b.Checksum = (b.Checksum << 8) | uint64(csbyte)
		}
	}
	return b, nil
}

func WriteBlock(fw *FrameWriter, b Block) error {
	// if EOS block is being written
	if b.BlockType == EOS {
		_, err := fw.Writer.Write([]byte{b.BlockType})
		return err
	}
	// build block header
	bytes := make([]byte, 0, 27)
	bytes = append(bytes, b.BlockType)
	if b.BlockType == BlockCodec {
		bytes = append(bytes, b.Codec)
	}
	bytes = binary.AppendUvarint(bytes, b.USize)
	bytes = binary.AppendUvarint(bytes, b.CSize)
	hasCCS := fw.Header.ChecksumMode&CompressedChecksum != 0
	hasUCS := fw.Header.ChecksumMode&UncompressedChecksum != 0
	if hasUCS && hasCCS {
		bytes = binary.BigEndian.AppendUint64(bytes, b.Checksum)
	} else if hasUCS || hasCCS {
		bytes = binary.BigEndian.AppendUint32(bytes, uint32(b.Checksum))
	}

	// write the bytes
	_, err := fw.Writer.Write(bytes)
	if err != nil {
		return fmt.Errorf("error in writing block - %s: %v", b, err)
	}
	return nil
}
