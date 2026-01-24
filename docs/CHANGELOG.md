# Change Log
All notable changes to this project will be documented in this file.
 
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
