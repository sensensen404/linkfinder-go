# linkfinder-go

This is a Go version of the [LinkFinder](https://github.com/GerbenJavado/LinkFinder) tool. It extracts URLs and paths from files and directories using a regular expression.

## Usage

### Command-Line Flags

- `-f <file>`: Specify a single input file.
- `-d <directory>`: Specify a directory to scan for files.
- `-o <output_file>`: Specify an output file to write results to.

### Example

```sh
linkfinder-go -f input.txt -o output.txt
linkfinder-go -d /path/to/directory -o output.txt
```

## Credits

Credit to [GerbenJavado/LinkFinder](https://github.com/GerbenJavado/LinkFinder) for the original idea and regex.
