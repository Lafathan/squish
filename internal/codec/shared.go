package codec

import (
	"fmt"
	"squish/internal/sqerr"
)

// codec IDs
const (
	RAW     = iota // 0
	RLE            // 1
	RLE2           // 2
	RLE3           // 3
	RLE4           // 4
	ZRLE           // 5
	LZSS           // 6
	LRLE           // 7
	LRLE2          // 8
	LRLE3          // 9
	LRLE4          // 10
	HUFFMAN        // 11
	MTF            // 12
	BWT            // 13
	DELTA          // 14
	XOR            // 15
	AUTO           // 16
)

// codec key map
var CodecMap = map[uint8]Codec{
	// lossless codecs
	RAW:  RAWCodec{},
	RLE:  RLECodec{byteLength: 1, lossless: true},
	RLE2: RLECodec{byteLength: 2, lossless: true},
	RLE3: RLECodec{byteLength: 3, lossless: true},
	RLE4: RLECodec{byteLength: 4, lossless: true},
	ZRLE: ZRLECodec{},
	LZSS: LZSSCodec{},
	// lossy codecs
	LRLE:  RLECodec{byteLength: 1, lossless: false},
	LRLE2: RLECodec{byteLength: 2, lossless: false},
	LRLE3: RLECodec{byteLength: 3, lossless: false},
	LRLE4: RLECodec{byteLength: 4, lossless: false},
	// entropy codecs
	HUFFMAN: HUFFMANCodec{},
	// transforms
	MTF:   MTFCodec{},
	BWT:   BWTCodec{},
	DELTA: DELTACodec{},
	XOR:   XORCodec{},
	// auto
	AUTO: &AUTOCodec{},
}

// codec string to codec ID map
var StringToCodecIDMap = map[string]uint8{
	"RAW":     RAW,
	"RLE":     RLE,
	"RLE2":    RLE2,
	"RLE3":    RLE3,
	"RLE4":    RLE4,
	"ZRLE":    ZRLE,
	"LZSS":    LZSS,
	"LRLE":    LRLE,
	"LRLE2":   LRLE2,
	"LRLE3":   LRLE3,
	"LRLE4":   LRLE4,
	"HUFFMAN": HUFFMAN,
	"MTF":     MTF,
	"BWT":     BWT,
	"DELTA":   DELTA,
	"XOR":     XOR,
	"AUTO":    AUTO,
}

// codec aliases
var CodecAliases = map[string]string{
	"DEFLATE": "LZSS-HUFFMAN",
	"BZIP":    "BWT-MTF-ZRLE-HUFFMAN",
}

// codec interface
type Codec interface {
	EncodeBlock(src []byte) (dst []byte, err error)
	DecodeBlock(src []byte) (dst []byte, err error)
	IsLossless() bool
}

// send a byte slice through a pipeline of encodings
func EncodePipeline(src []byte, pipeline []uint8) ([]byte, error) {
	var (
		temp = append([]byte(nil), src...)
		err  error
	)
	for i := range len(pipeline) {
		currentCodec, ok := CodecMap[pipeline[i]]
		if !ok {
			return temp, sqerr.New(sqerr.Unsupported, "unsupported codec ID")
		}
		temp, err = currentCodec.EncodeBlock(temp)
		if err != nil {
			return temp, sqerr.CodedError(err, sqerr.Internal, fmt.Sprintf("failed to encode block of data with codec %d", currentCodec))
		}
	}
	return temp, nil
}

// send a byte slice backwards through a pipeline of decodings
func DecodePipeline(src []byte, pipeline []uint8) ([]byte, error) {
	var (
		temp = append([]byte(nil), src...)
		err  error
	)
	for i := len(pipeline) - 1; i >= 0; i-- {
		currentCodec, ok := CodecMap[pipeline[i]]
		if !ok {
			return temp, sqerr.New(sqerr.Unsupported, "unsupported codec ID")
		}
		temp, err = currentCodec.DecodeBlock(temp)
		if err != nil {
			return temp, sqerr.CodedError(err, sqerr.Internal, fmt.Sprintf("failed to decode block of data with codec %d", currentCodec))
		}
	}
	return temp, nil
}

// byte histogram function
func histogram[T byte | int32](data []T, freq []int32) {
	clear(freq)
	for _, v := range data {
		freq[v]++
	}
}

// create cumulative sum in-place
func cumSum(s []int32) {
	var (
		sum int32 = 0
		val int32
	)
	for i := range len(s) {
		val = s[i]
		s[i] = sum
		sum += val
	}
}

// grow a slice
func grow32[T int32 | uint32](slice []T, length int) []T {
	if cap(slice) < length {
		return make([]T, length)
	}
	s := slice[:length]
	clear(s)
	return s
}

// clamp a float64 value
func clampFloat[T float32 | float64](f, lo, hi T) T {
	if f < lo {
		return lo
	}
	if f > hi {
		return hi
	}
	return f
}

// absolute byte delta
func absByteDiff(a, b byte) byte {
	if a >= b {
		return a - b
	}
	return b - a
}
