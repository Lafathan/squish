package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

type Block struct {
	BlockType uint8   // 0x00 EOS, 0x01 Default codec, 0x02 Block codec
	Codec     []uint8 // only used if BlockType > 0
	USize     uint64  // uncompressed size
	CSize     uint64  // compressed size
	Checksum  uint64  // checksum value 4 bytes for uncompressed, 4 bytes for compressed
}

func (b *Block) valid() error {
	if (b.BlockType != EOS) && (b.BlockType != DefaultCodec) && (b.BlockType != BlockCodec) {
		return errors.New("invalid block type found")
	}
	if b.USize > MaxBlockSize {
		return errors.New("invalid block size found")
	}
	return nil
}

func (block1 Block) equal(block2 Block) bool {
	a := block1.BlockType == block2.BlockType
	b := block1.USize == block2.USize
	c := block1.CSize == block2.CSize
	d := block1.Checksum == block2.Checksum
	e := true
	for i := range block1.Codec {
		e = block1.Codec[i] == block2.Codec[i]
		if !e {
			return false
		}
	}
	return a && b && c && d && e
}

func (b Block) String() string {
	s := fmt.Sprintf("BlockType: %d\n", b.BlockType)
	s += fmt.Sprintf("Codec:     %d\n", b.Codec)
	s += fmt.Sprintf("USize:     %d\n", b.USize)
	s += fmt.Sprintf("CSize:     %d\n", b.CSize)
	s += fmt.Sprintf("Checksum:  %016x\n", b.Checksum)
	return s
}

func readBlock(fr *frameReader) (Block, error) {
	var b Block
	var err error
	// get block type
	b.BlockType, err = fr.ReadByte()
	if err != nil {
		return b, fmt.Errorf("error in reading block type: %w", err)
	}
	// return if EOS block
	if b.BlockType == EOS {
		return b, nil
	}
	// read the number of codec if there is a block specific one
	codecs := byte(0)
	if b.BlockType == BlockCodec {
		codecs, err = fr.ReadByte()
		if err != nil {
			return b, fmt.Errorf("error in reading block codecs: %w", err)
		}
	}
	// read the order of the codecs
	b.Codec, err = fr.ReadBytes(int(codecs))
	if err != nil {
		return b, fmt.Errorf("error in reading block codec list: %w", err)
	}
	// read and assign the varint sizes
	b.USize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, fmt.Errorf("error in reading block uncompressed size: %w", err)
	}
	b.CSize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, fmt.Errorf("error in reading block compressed size: %w", err)
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
			return b, fmt.Errorf("error in reading block checksum: %w", err)
		}
		for _, csbyte := range cs {
			b.Checksum = (b.Checksum << 8) | uint64(csbyte)
		}
	}
	return b, nil
}

func writeBlock(fw *frameWriter, b Block) error {
	// if EOS block is being written
	if b.BlockType == EOS {
		_, err := fw.writer.Write([]byte{b.BlockType})
		return err
	}
	// build block header
	bytes := make([]byte, 0, 27)
	bytes = append(bytes, b.BlockType)
	if b.BlockType == BlockCodec {
		bytes = append(bytes, byte(len(b.Codec)))
		bytes = append(bytes, b.Codec...)
	}
	bytes = binary.AppendUvarint(bytes, b.USize)
	bytes = binary.AppendUvarint(bytes, b.CSize)
	hasCCS := fw.header.ChecksumMode&CompressedChecksum != 0
	hasUCS := fw.header.ChecksumMode&UncompressedChecksum != 0
	if hasUCS && hasCCS {
		bytes = binary.BigEndian.AppendUint64(bytes, b.Checksum)
	} else if hasUCS || hasCCS {
		bytes = binary.BigEndian.AppendUint32(bytes, uint32(b.Checksum))
	}
	// write the bytes
	_, err := fw.writer.Write(bytes)
	if err != nil {
		return fmt.Errorf("error in writing block - %s: %w", b, err)
	}
	return nil
}
