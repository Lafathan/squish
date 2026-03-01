package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"squish/internal/sqerr"
)

type Block struct {
	BlockType uint8   // 0x00 EOS, 0x01 Default codec, 0x02 Block codec
	Codec     []uint8 // only used if BlockType == 1
	USize     uint64  // uncompressed size, written to file as uvarint
	CSize     uint64  // compressed size, written to file as uvarint
	Checksum  uint64  // checksum value 4 bytes for uncompressed, 4 bytes for compressed
}

func (b *Block) valid() error {
	// checks validity of a block
	// determined by valid block type byte and a block size that isn't too large
	if (b.BlockType != EOS) && (b.BlockType != DefaultCodec) && (b.BlockType != BlockCodec) {
		return sqerr.New(sqerr.Corrupt, "invalid block type found")
	}
	if b.USize > MaxBlockSize {
		return sqerr.New(sqerr.Corrupt, "invalid block size found")
	}
	return nil
}

func (block1 Block) equal(block2 Block) bool {
	// checks if two blocks are the same
	a := block1.BlockType == block2.BlockType
	b := block1.USize == block2.USize
	c := block1.CSize == block2.CSize
	d := block1.Checksum == block2.Checksum
	e := bytes.Equal(block1.Codec, block2.Codec)
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
	// read in the details of a block
	var (
		b   Block
		err error
	)
	// get block type and return if EOS
	b.BlockType, err = fr.ReadByte()
	if err != nil {
		return b, fmt.Errorf("failed to read block type: %w", err)
	}
	if b.BlockType == EOS {
		return b, nil
	}
	// if the codec is block specific (blocktype codec), read in the codecs
	if b.BlockType == BlockCodec {
		codecs, err := fr.ReadByte()
		if err != nil {
			return b, fmt.Errorf("failed to read block codec number: %w", err)
		}
		b.Codec, err = fr.ReadBytes(int(codecs))
		if err != nil {
			return b, fmt.Errorf("failed to read block codec list: %w", err)
		}
	}
	// read in the varint stored sizes into uint64 values
	b.USize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, fmt.Errorf("failed to read block uncompressed size: %w", err)
	}
	b.CSize, err = binary.ReadUvarint(fr)
	if err != nil {
		return b, fmt.Errorf("failed to read block compressed size: %w", err)
	}
	// read in the checksum data
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
			return b, fmt.Errorf("failed to read block checksum: %w", err)
		}
		for _, csbyte := range cs {
			b.Checksum = (b.Checksum << 8) | uint64(csbyte)
		}
	}
	return b, nil
}

func writeBlock(fw *frameWriter, b Block) error {
	// write a block to a frame
	if b.BlockType == EOS {
		// if EOS block is being written, break
		_, err := fw.writer.Write([]byte{b.BlockType})
		if err != nil {
			return fmt.Errorf("failed to write EOS block: %w", err)
		}
		return nil
	}
	// build the block header
	// size = block type + codecs + codec list (1 to 255, usually less than 4) +
	// ucompressed size (8) + compressed size (8) + checksum + (8)
	bytes := make([]byte, 0, 33)
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
	_, err := fw.writer.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write block: %w", err)
	}
	return nil
}
