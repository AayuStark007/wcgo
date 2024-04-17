# wcgo

`wcgo` is a Go implementation of the classic `wc` (word count) command-line tool, providing a fast and efficient way to count words, lines, and characters in text files. Inspired by the UNIX `wc` utility, `wcgo` aims to offer similar functionality with the added benefits of Go's performance and concurrency features.

## Features

- **Line Count:** Count the number of lines in a text file.
- **Word Count:** Count the number of words in a text file.
- **Character Count:** Count the number of characters in a text file.
- **Multiple Files:** Supports processing multiple text files.
- **Standard In:** Supports Standard In (> and pipe works)
- **Concurrency[WiP]:** Utilize Go's concurrency model for handling multiple files efficiently.

## Getting Started

### Prerequisites

Ensure you have Go installed on your system. `wcgo` requires Go version 1.13 or later. You can check your Go version by running:

```bash
go version
```

### Installing

Clone the `wcgo` repo to your local machine:

```bash
git clone https://github.com/AayuStark007/wcgo.git
cd wcgo 
```

### Build

Build the binary with: 

```bash
go build
```

The executable `wcgo` is generated in the current directory.

### Running `wcgo`

Run `./wcgo --help` to view the full help options

```bash
$ ./wcgo --help
A Go implementation of wc to print newline, word, and byte counts for each file

Usage:
  wcgo [file]... [flags]

Flags:
  -c, --bytes   print the byte counts
  -m, --chars   print the character counts
  -d, --debug   debug mode
  -h, --help    help for wcgo
  -l, --lines   print the newline counts
  -w, --words   print the word counts
```

Additionally you can specify the filename:
```bash
./wcgo examples/test.txt
```

### Usage Examples

Count words in `example.txt`

```bash
./wcgo -w example.txt
```

Count lines in `example.txt`

```bash
./wcgo -l example.txt
```

Count words, lines and bytes via Standard Input

```bash
cat example.txt | ./wcgo
```

Process multiple files

```bash
./wcgo file1.txt file2.txt file3.txt
```

### Limitations

Since, this project is in development, some features of classic `wc` are yet to be implemented:

- Support for passing `-` as file to enable reading from stdin
- Support for fixed with printing (currently output is space separated)

Since, we are implementing this in Go, this allows scope for utilizing some language features for more performance:

- Using Goroutines for processing multiple files concurrently.
- Using buffered io to handle large files without using up too much memory

### License

`wcgo` is open-sourced under the MIT License. See the LICENSE file for more details.

### Acknowledgments
- This project is inspired by the original UNIX wc command.
- Motivation was from the challenges posted at [Coding Challenges FYI](https://codingchallenges.fyi/).

#### TODO
- benchmarks
- fixes for word counting in binary files
- performance tuning
- efficient concurrency (vs goroutine per file)
