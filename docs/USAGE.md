
# Squish User Guide

This guide explains how to use `squish` (the CLI), what behavior to expect.
If you want the binary format specification, see: `docs/format.md`.

---

## Table of Contents
- [What Squish is](#what-squish-is)
- [Installation](#installation)
- [Quickstart](#quickstart)
- [Core concepts](#core-concepts)
- [Commands](#commands)
- [Pipelines and codecs](#pipelines-and-codecs)
- [Checksums and verification](#checksums-and-verification)
- [Block sizing](#block-sizing)
- [Working with stdin/stdout](#working-with-stdinstdout)
- [File naming and extensions](#file-naming-and-extensions)
- [Exit codes](#exit-codes)

---

## What Squish is

Squish is a command-line compression utility.
It supports:
- compressing and decompressing generic binary data
- optional integrity checking (checksums)
- optional analysis/inspection of compressed streams
- (optional) codec pipelines (e.g. `rle-huffman`)

### What squish is NOT
Squish does *not* currently aim to:
- beat modern general-purpose compressors on ratio
- provide encryption
- guarantee stable output across versions
- compressing already-compressed/high-entropy data may increase size

---

## Installation

### From source
```bash
go install squish/cmd/squish@latest
squish --version
squish --help
```
## Quickstart
### Compress a file
```bash
squish enc -o output.sqz input.bin
```
### Decompress a file
```bash
squish dec -o output.bin input.sqz
```
### Use a pipeline
```bash
squish enc -o output.sqz --codec rle-huffman input.bin
```

## Core concepts

### Input and output model
Squish reads bytes from a file (or stdin) and writes a compressed stream (to a file or stdout).
Decompression reverses the process.

#### Frames and blocks (high-level)
A `sqz` stream contains a header and one or more blocks.
Blocks allow streaming: Squish does not need the whole file in memory.
Block size affects compresstion ratio and memory usage.
For the exact layout, see docs/format.md.

#### Determinism
Given the same input and flags, output is byte-identical within the same squish version/build.

### Commands
#### squish enc
Compresses input into a `.sqz` stream.
##### Usage
```bash
squish enc -o [output] [flags] [input]
```
##### Examples
```bash
squish enc -o data.sqz data.bin
squish enc -o data.sqz --codec huffman data.bin
squish enc -o data.sqz --codec rle-huffman --blocksize 256KiB data.bin
```
##### Behavior
If -o is omitted, output goes to stdout.
If input is omitted, input is read from stdin.

By default, Squish uses DEFLATE with a 25KiB block size and no checksum integrity checks. Additionally, if the output already exists, squish will overwrite it.

##### Common flags

`--codec <pipeline>`: Selects codec(s) used for compression.
`--blocksize <n>`: Sets block size (see Block sizing).
`--checksum <mode>`: Controls checksum behavior (see Checksums).

#### squish dec
Decompresses a `.sqz` stream back to raw bytes.
##### Usage
```bash
squish dec -o [output] [flags] [input]
```
##### Examples
```bash
squish dec data.sqz -o data.bin
squish dec data.sqz > data.bin
```
##### Behavior
If a lossy codec is present, uncompressed checksum verification is disabled.

If the stream is truncated/corrupt, squish will return with a corrupt exit code.

### Pipelines and codecs
A pipeline is a list of codecs to be applied or that have been applied to a stream of data. This can include anywhere from a single codec (a pipeline of one), up to 255 codecs. The pipeline describes the order the codecs are applied, left-to-right, with the reverse being applied, right-to-left, during decompression.

The dash/hyphen symbol is used to delineate between codecs with no whitespace. Codec names are not case sensitive. Unknown codecs return with an "unknown codec" error and unsupported exit code. 

Run the `squish enc --list-codecs` command for a list of canonical names of available codecs to use in pipelines.

Example:
```bash
--codec "rle-huffman"
--codec "LRLE2-RLE-LZSS-HUFFMAN"
```
#### Available codecs
- RAW - Pass-through mode: stores the data as-is with only Squish framing/metadata. Useful as a baseline for benchmarking and for verifying the container/IO path without compression effects.
- RLE - Run-Length Encoding: replaces long runs of the _same_ value with (value, count). Works best on highly repetitive, low-entropy data (e.g., zero-filled regions, simple masks, flat-color pixels).
- LRLE - Lossy Run-Length Encoding: like RLE, but allows values within a tolerance to be treated as “the same,” encoding them as a single representative value + count. Best for “almost constant” signals (noisy sensors, gently varying channels, lightly dithered imagery) when small, controlled loss is acceptable.
- HUFFMAN - Entropy coding: assigns shorter bit codes to more frequent symbols and longer codes to rare ones. Great when byte values have a skewed distribution (text-like data, structured binaries, outputs of other transforms). Typically helps more as a second-stage codec.
- LZSS - Dictionary-based (LZ77-family): encodes repeated sequences by referencing earlier occurrences with (offset, length) pairs, falling back to literals when no good match exists. Strong general-purpose compressor for data with repeated substrings/patterns (text, logs, structured formats).
- DEFLATE - Convenience alias for the LZSS-HUFFMAN pipeline.

#### Lossy codecs
**Data loss warning**: Lossy codecs (e.g., LRLE*) intentionally change the data to improve compression ratio. Files compressed with a lossy codec will not decompress to the original bytes, and should not be used for data where exact recovery matters. If you need byte-for-byte fidelity, use lossless codecs only.

#### Decode order
Compression applies left-to-right; decompression reverses that.

### Checksums and verification
Squish validates block boundaries using stored compressed/uncompressed sizes. CRC32 checksums (optional) provide integrity validation of the bytes, not just the lengths. The user has the option to apply checksum validation to the uncompressed data, the compressed data, or both. These validation checks are stored and applied to each and every block.
```bash
--checksum u  # applied to uncompressed data
--checksum c  # applied to compressed data
--checksum uc # applied to both compressed and uncompressed data
```
In the case that a checksum fails, squish returns with a corrupt error code and stops decompressing. Partial output may have been written at this point. At this time, there is no option to continue with the decompression or try to recover any further data. If decompressing to a file, consider writing to a temprorary file and renaming on success to avoid overwriting with partial data on corruption.

### Block sizing
Squish encodes data in chunks of a consistent size called blocks. This allows the data to be streamed through squish and not opened entirely in memory allowing for very large files to be compressed.

Smaller blocks: better streaming, more overhead, likely worse compression ratio
larger blocks: better compression ratio, more memory usage

#### Accepted formats
The accepted formats are an integer followed by a valid, case-sensitive, unit of byte size.
The valid units are as follows
- KiB - Kibibyte (1,024 bytes)
- KB - Kilobyte (1,000 bytes)
- MiB - Mebibyte (1,048,576 bytes)
- MB -  Megabyte (1,000,000 bytes)
- B - Byte
```bash
--blocksize 256KiB
--blocksize 1MiB
--blocksize 4096B
```

### Working with stdin/stdout
Squish defaults to stdin and stdout when not given any -o, --output, or [input] values. This makes it extremely easy to use in conjunction with commands whose output you want to compress/decompress.
```bash
cat input.bin | squish enc > out.sqz
cat out.sqz | squish dec > restored.bin
```

### File naming and extensions
While squish does not care what you ask it to encode/decode, the convention is to give any output written to disk the recommended extension: `.sqz`

### Exit codes
```
0: success
1: usage error / bad flags
2: I/O error
3: corrupt input / checksum mismatch
4: unsupported codec / version
5: internal error
```
