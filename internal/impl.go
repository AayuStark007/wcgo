//go:build linux
// +build linux

package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode"

	"golang.org/x/sys/unix"

	"github.com/spf13/cobra"
)

const CHUNK_SIZE = 16 * 1024

var (
	Bytes bool
	Lines bool
	Words bool
	Chars bool
	Debug bool
)

type WCContext struct {
	sync.RWMutex

	flagBytes bool
	flagLines bool
	flagWords bool
	flagChars bool
	flagNone  bool
	flagStdin bool

	files []string
	index uint16
	fd    *os.File // current open fd
	done  bool     // whether current fd is closed or EOF

	result wcresult
}

type wcresult struct {
	sync.RWMutex
	words int32
	lines int32
	bytes int64
	chars int64
	err   error
}

// TODO: move to bitfield based flags

func New(files []string, bytes, lines, words, chars bool) (*WCContext, error) {
	var flagStdin bool = false
	if len(files) <= 0 {
		flagStdin = true
	}

	var fd *os.File
	var err error

	if flagStdin {
		fd = os.Stdin
	} else {
		fd, err = os.Open(files[0])
	}

	return &WCContext{
		flagBytes: bytes,
		flagLines: lines,
		flagWords: words,
		flagChars: chars,
		flagNone:  !bytes && !lines && !words && !chars,
		flagStdin: flagStdin,

		files: files,
		index: 0,
		fd:    fd,
		done:  false,
		result: wcresult{
			words: 0,
			lines: 0,
			bytes: 0,
			chars: 0,
			err:   nil,
		},
	}, err
}

func (ctx *WCContext) currentFd() *os.File {
	ctx.RLock()
	defer ctx.RUnlock()

	return ctx.fd
}

func (ctx *WCContext) Compute() {
	ctx.result.Lock()
	defer ctx.result.Unlock()
	ctx.computeInternal()
}

func (ctx *WCContext) computeInternal() {
	// since stdin is not seekable, we need to compute all the counts at once

	// use bufio to read 16384 bytes at a time
	// bytes => count num of bytes read and add
	// lines => count each \n byte and add
	// words => our routine can be used
	// chars => read byte slice as rune and count
	fd := ctx.currentFd()
	unix.Fadvise(int(fd.Fd()), 0, 0, unix.FADV_SEQUENTIAL)
	reader := bufio.NewReaderSize(fd, CHUNK_SIZE)
	buff := make([]byte, CHUNK_SIZE)
	var countBytes int64 = 0
	var countLines int32 = 0
	var countWords int32 = 0
	var countChars int64 = 0

	word := false

	for {
		// clear(buff)
		numBytes, err := reader.Read(buff)
		if err == io.EOF || numBytes == 0 {
			if word {
				countWords++
			}
			ctx.result.bytes = countBytes
			ctx.result.lines = countLines
			ctx.result.words = countWords
			ctx.result.chars = countChars

			return
		}

		countBytes += int64(numBytes)

		reader := bytes.NewReader(buff[:numBytes])
		countLines += lineCount(reader)

		// special handling in case we end buffer in middle of a word
		reader.Seek(0, io.SeekStart)
		count, wrd := wordCount(reader, word)
		word = wrd
		countWords += count

		reader.Seek(0, io.SeekStart)
		countChars += charCount(reader)
	}
}

func (ctx *WCContext) currentFile() string {
	if ctx.flagStdin {
		return ""
	}
	return ctx.files[ctx.index]
}

func (ctx *WCContext) String() string {
	if ctx.result.err != nil {
		return ctx.result.err.Error()
	}

	// TODO: print for all files, along with total
	// TODO: not the correct approach, cannot handle flag combinations (use string buffer)
	ctx.result.RLock()
	defer ctx.result.RUnlock()

	if ctx.flagBytes {
		return fmt.Sprintf("%d %s\n", ctx.result.bytes, ctx.currentFile())
	}

	if ctx.flagLines {
		return fmt.Sprintf("%d %s\n", ctx.result.lines, ctx.currentFile())
	}

	if ctx.flagWords {
		return fmt.Sprintf("%d %s\n", ctx.result.words, ctx.currentFile())
	}

	if ctx.flagNone {
		return fmt.Sprintf("%d %d %d %s\n", ctx.result.lines, ctx.result.words, ctx.result.bytes, ctx.currentFile())
	}

	if ctx.flagChars {
		return fmt.Sprintf("%d %s\n", ctx.result.chars, ctx.currentFile())
	}

	return ""
}

func (ctx *WCContext) Close() error {
	ctx.Lock()
	defer ctx.Unlock()

	ctx.fd.Close()
	ctx.done = true

	return nil
}

func Handle(cmd *cobra.Command, args []string) {
	ctx, err := New(args, Bytes, Lines, Words, Chars)
	if err != nil {
		panic(err)
	}
	defer ctx.Close()

	ctx.Compute()

	fmt.Print(ctx.String())
}

// TODO: this is not an exhaustive check,
// need to define what byte constitutes a valid candidate for word and change the logic
func isSpecial(char byte) bool {
	if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
		return true
	} else {
		return false
	}
}

// lineCount gets the number of lines in the currently open file
func lineCount(reader io.ByteReader) int32 {
	var count int32 = 0

	for {
		if byt, err := reader.ReadByte(); err == io.EOF {
			return count
		} else if err != nil {
			return 0
		} else if byt == '\n' {
			count++
		}
	}
}

// wordCount gets the number of words in the currently open file
func wordCount(reader io.ByteReader, word bool) (int32, bool) {
	var count int32 = 0

	for {
		if byt, err := reader.ReadByte(); err == io.EOF {
			return count, word
		} else {
			if isSpecial(byt) && word {
				count++
				word = false
			} else if !isSpecial(byt) && !word {
				word = true
			}
		}
	}
}

func charCount(reader io.RuneReader) int64 {
	var count int64 = 0

	for {
		if r, _, err := reader.ReadRune(); err == io.EOF {
			return count
		} else if err != nil || r == unicode.ReplacementChar {
			return 0
		} else {
			count++
		}
	}
}
