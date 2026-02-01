# Change Log
All notable changes to this project will be documented in this file.

## [0.2.0] - 2026-01-31

Automagic encoding is here!

### Added
- Added `-codec AUTO` flag to algorithmically decide the best encoding pipeline for your dataset
    + Efficiently probes a subset of your data to test possible encodings 
    + Applies per-block unique encoding so best pipeline is chosen for each chunk of your data

### Changed 
- Increased max blocksize from 65KB to 16.8MB
- Increased default blocksize from 25KiB to 128KiB
- Decreased size of executable by ~1/3

### Fixed
- 20% less memory usage when decoding Huffman
- Less memory usage for Run Length Encoding as well
- Fixed not recognizing non-upper-case `deflate` codec.
    
## [0.1.1] - 2026-01-25

Performance improvement for LZSS.

### Changed
- LZSS encoding saw massive performance improvement using hash look-ups for substring matches
    + No changes to LZSS API
 
## [0.1.0] - 2026-01-24
  
Initial release
 
### Added
- Added `sqz` frame format for encoding and decoding with following available codecs
    + RAW (passthrough)
    + Run Length Encoding (8 variations of lossiness and byte-stride)
    + HUFFMAN (canonical)
    + Lempel–Ziv–Storer–Szymanski (LZSS)
- Added ability to pipe multiple encodings together
- Added integrity checks of payload size and CRC32 checksums

### Changed
  
- Everything

### Fixed
 
- N/A
