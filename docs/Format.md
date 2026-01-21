# Squish `.sqz` File Format Specification (Draft)

> Status: **DRAFT / WORKING DOC**  
> This document describes the on-disk format for Squish-compressed streams with the `.sqz` extension.
> Some fields and semantics are intentionally left as placeholders so the format can evolve.

---

## 1. Goals and non-goals

### Goals
- **Streamable**: decode sequentially without seeking.
- **Block-based**: enables bounded memory usage and partial recovery.
- **Codec-agnostic**: frames and blocks declare how to decode their payload.
- **Integrity options**: checksum fields are well-defined and unambiguous.

### Non-goals
- Random access (unless explicitly enabled by an index feature in a future version).
- Encrypting or authenticating content (checksums are not security).
- Forward/backward-compatible: At this early stage, new features might be added that break old decoders.

---

## 2. Terminology

- **Stream**: the full `.sqz` file.
- **Frame**: the stream container structure (header + blocks).
- **Block**: the unit of compression and integrity checking.
- **Codec**: an algorithm used to encode/decode block payloads (e.g., Huffman, RLE, LZSS).
- **Pipeline**: multiple codecs applied in sequence to a block.
- **Payload**: the encoded bytes for a block (after applying codecs).
- **Plaintext / raw bytes**: the original bytes for a block prior to encoding.

---

## 3. Conventions

### 3.1 Byte order (endianness)
All multi-byte integer fields are **big-endian** unless stated otherwise.

### 3.2 Integer types
- `uint8`  : unsigned 8-bit
- `uint16` : unsigned 16-bit
- `uint32` : unsigned 32-bit
- `uint64` : unsigned 64-bit

### 3.3 Sizes and limits (placeholders)
These are recommended initial limits:
- Max block raw size: `<= 64 MiB`
- Max block payload size: `<= raw size + overhead`
- Max pipeline length: `<= 255` codecs

---

## 4. High-level file layout

A `.sqz` stream is:

| File Header |
|---|
| Block 0 |
| Block 1 |
| ... |

A decoder reads the header, then decodes blocks sequentially until it reaches an explicit end marker block.

---

## 5. File Header

### 5.1 Header overview


| Field | Type / Size |
|----------------------|-----------------------------|
| Magic | 3 bytes |
| Flags | byte |
| Checksum mode | byte |
| Codec Count | uint8 |
| Codec List | [Codec Count]uint8 |

#### 5.2 Magic (3 bytes)
Identifies a Squish stream.
- `"SQZ"`

#### 5.3 Flags (byte)
Bitfield controlling baseline behaviors. Placeholder bits:
- `bit 0`: Not used
- `bit 1`: Not used
- ...
- `bit 7`: Not used

#### 5.4 Checksum mode (uint8)
Integer describing checksum protocals to apply to each block during decoding:
- `0`: No checksums used
- `1`: Uncompressed data is validated via crc32 checksum
- `2`: Compressed data is validated via crc32 checksum
- `3`: Both uncompressed and compressed data is validated via crc32 checksum

#### 5.5 Codec List Length (uint8)
Length of codecs in the pipeline

### 5.6 Codec List ([]uint8)
List of uint8 values representing the codec IDs in the pipline in order they were applied during encoding

---

## 6. Block format

### 6.1 Block overview

Each block begins with a fixed header:

| Field | Type / Size |
|---|---|
| Block Type | uint8 |
| Codec Count | uint8 |
| Codecs | [Codec Count]uint8 |
| Raw Size | varint64 |
| Payload Size | varint64 |
| Optional Checksums | 0, 8, or 16 bytes (depending on checksum mode) |


### 6.2 Block Type (uint8)
Defines how to interpret the block and whether it emits bytes.

- `0x00` End-of-stream marker (no payload; marks the end of the block)
- `0x01` Frame encoded data block (normal)
- `0x02` Block encoded data block (uses a specific encoding and may not adhere to frame encoding)

### 6.3 Codec Count (uint8)
Number of codecs in the pipeline. Only present when block type is `0x02`.

### 6.4 Codecs ([Codec Count]uint8)
A list of codec IDs used to encode the block. Only present when block type is `0x02`.

### 6.5 Raw Size (varint64)
Number of bytes produced by decoding this block.  

### 6.6 Payload Size (varint64)
Number of bytes following the header that belong to the payload.

### 6.7 Optional Checksums (variable)
If enabled, crc32 checksums appear in the following order:

1. `raw_crc32`  (uint32) — CRC of raw bytes (after fully decoded)
2. `payload_crc32` (uint32) — CRC of payload bytes (as stored)

---

## 7. Codec registry

Codec IDs are `uint8`.

- `0x00` RAW (passthrough)
- `0x01` RLE (lossless)
- `0x02` RLE2 (lossless, 2-byte stride)
- `0x03` RLE3 (lossless, 3-byte stride)
- `0x04` RLE4 (lossless, 4-byte stride)
- `0x05` LRLE (lossy)
- `0x06` LRLE2 (lossy, 2-byte stride)
- `0x07` LRLE3 (lossy, 3-byte stride)
- `0x08` LRLE4 (lossy, 4-byte stride)
- `0x09` HUFFMAN
- `0x0A` LZSS

Additional codecs are defined via codec aliases when they can be represented by a pipeline.
- DEFLATE -> LZSS and HUFFMAN

---

## 8. End-of-stream behavior

A stream terminates either by reading a block with `Block Type = EOS`
