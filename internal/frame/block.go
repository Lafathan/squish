package frame

import (
	"encoding/binary"
	"errors"
)

type Block struct {
	BlockType      uint8  // 0x00 EOS, 0x01 Default codec, 0x02 Block codec
	Codec          uint8  // only used if BlockType == 0x02
	USize          uint64 // uncompressed size
	CSize          uint32 // compressed size
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
	// read in the blockType first to make sure there is more to read
	var b Block
	blockType, err := fr.ReadBytes(1)
	if err != nil {
		return b, err
	}
	if blockType[0] == 0 {
		return b, nil // EOS block type encountered
	}
	bytes, err := fr.ReadBytes(22)
	if err != nil {
		return b, err
	}

	// assign values to the block
	b.BlockType = blockType[0]
	b.Codec = bytes[0]
	b.USize = binary.BigEndian.Uint64(bytes[1:9])
	b.CSize = binary.BigEndian.Uint32(bytes[9:13])
	b.ChecksumMethod = bytes[13]
	b.Checksum = binary.BigEndian.Uint64(bytes[14:22])

	return b, nil
}

func WriteBlock(fw *FrameWriter, b Block) error {
	// if EOS block is being written
	if b.BlockType == 0 {
		err := fw.Writer.WriteByte(b.BlockType)
		if err != nil {
			return err
		}
		return nil
	}
	// build block header
	bytes := []byte{}
	bytes = append(bytes, b.BlockType, b.Codec)
	bytes = binary.BigEndian.AppendUint64(bytes, b.USize)
	bytes = binary.BigEndian.AppendUint32(bytes, b.CSize)
	bytes = append(bytes, b.ChecksumMethod)
	bytes = binary.BigEndian.AppendUint64(bytes, b.Checksum)

	// write the bytes
	_, err := fw.Writer.Write(bytes)
	return err
}
