package cli

const decHelp = `
squish dec - decompress a .sqz stream into original bytes

USAGE:
  squish dec [input] [flags]

FLAGS:
  -o, --output <path|->    Output file (default: '-')

EXAMPLES:
  squish dec ./file.sqz -o ./file
  squish dec ./file.sqz -o -
  cat file.sqz | squish dec > file
`
