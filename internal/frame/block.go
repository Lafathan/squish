package frame

import (
	"encoding/binary"
	"errors"
)

type Block struct {
	BlockType      uint8  // 0x00 EOS, 0x01 Default codec, 0x02 Block codec
	Codec          uint8  // only used if BlockType == 0x02
	USize          uint64 // uncompressed size
	CSize          uint64 // compressed size
	ChecksumMethod uint8  // Checksum - uncompressed (0x01), compressed (0x02), both, or none
	Checksum       uint64 // checksum value 4 bytes for uncompressed, 4 bytes for compressed
}

func (b *Block) Valid() error {
	if b.BlockType >= 3 {
		return errors.New("invalid block type found")
	}
	if b.CSize > MaxBlockSize {
		return errors.New("invalid block size found")
	}
	if b.ChecksumMethod > 3 {
		return errors.New("invalid checksum method found")
	}
	return nil
}

func ReadBlock(fr *FrameReader) (Block, error) {
	var b Block
	var err error

	// get block type
	b.BlockType, err = fr.ReadByte()
	if err != nil {
		return b, err
	}

	// return if EOS block
	if b.BlockType == EOSCodec {
		return b, nil
	}

	// read the codec if there is a block specific one
	if b.BlockType == BlockCodec {
		b.Codec, err = fr.ReadByte()
		if err != nil {
			return b, err
		}
	}

	// read and assign the varint sizes
	b.USize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, err
	}
	b.CSize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, err
	}

	// read the checksum method
	b.ChecksumMethod, err = fr.ReadByte()
	if err != nil {
		return b, err
	}

	// read the checksum data according to the method
	byteLength := 0
	if b.ChecksumMethod&UncompressedChecksum != 0x00 {
		byteLength += ChecksumSize
	}
	if b.ChecksumMethod&CompressedChecksum != 0x00 {
		byteLength += ChecksumSize
	}
	if byteLength > 0 {
		cs, err := fr.ReadBytes(byteLength)
		if err != nil {
			return b, err
		}
		for _, csbyte := range cs {
			b.Checksum = (b.Checksum << 8) | uint64(csbyte)
		}
	}
	return b, nil
}

func WriteBlock(fw *FrameWriter, b Block) error {
	// if EOS block is being written
	if b.BlockType == 0 {
		_, err := fw.Writer.Write([]byte{0})
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
	bytes = append(bytes, b.ChecksumMethod)
	hasUCS := b.ChecksumMethod&UncompressedChecksum != 0
	hasCCS := b.ChecksumMethod&CompressedChecksum != 0
	if hasUCS && hasCCS {
		bytes = binary.BigEndian.AppendUint64(bytes, b.Checksum)
	} else if hasUCS || hasCCS {
		bytes = binary.BigEndian.AppendUint32(bytes, uint32(b.Checksum))
	}

	// write the bytes
	_, err := fw.Writer.Write(bytes)
	return err
}
