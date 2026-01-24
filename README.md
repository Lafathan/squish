# squish

`squish` is a small Go-based compression/decompression utility that writes a custom `.sqz` frame format with pluggable codecs (RAW, RLE, and Huffman). It can stream to and from files or stdin/stdout and supports optional checksums and block sizing.

## Features

- Encode data into `.sqz` frames with a configurable codec pipeline.
- Decode `.sqz` streams back to original bytes.
- Stream input/output via stdin/stdout for easy piping.
- Optional checksums for compressed and/or uncompressed blocks.

## Build

```sh
go build ./cmd/squish
```

## Usage

### General help

```sh
./squish -h
```

### List available codecs

```sh
./squish enc --list-codecs
```

### Encode

```sh
./squish enc -codec RLE-HUFFMAN -o ./output.sqz ./input.txt
./squish enc -codec RAW --blocksize 256KiB -o ./output.sqz -
```

### Decode

```sh
./squish dec -o ./output.txt ./output.sqz
./squish dec -o - ./output.sqz > ./output.txt
```

## Flags

### `enc`

- `-codec`: codec pipeline (e.g. `RLE-HUFFMAN`, default DEFLATE)
- `--blocksize`: block size (e.g. `256KiB`, `1MiB`, default 25KiB)
- `--checksum`: checksum mode (`u`, `c`, or `uc`, default None)
- `-o, --output`: output path (`-` for stdout, default `-`)
- `--list-codecs`: list supported codecs and exit

### `dec`

- `-o, --output`: output path (`-` for stdout, default `-`)
