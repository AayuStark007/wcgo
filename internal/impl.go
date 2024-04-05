package internal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode"

	"github.com/spf13/cobra"
)

var (
	Bytes bool
	Lines bool
	Words bool
	Chars bool
)

type WCContext struct {
	sync.RWMutex

	flagBytes bool
	flagLines bool
	flagWords bool
	flagChars bool
	flagNone  bool

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
	if len(files) <= 0 {
		return nil, errors.New("no file name provided")
	}

	fd, err := os.Open(files[0])
	return &WCContext{
		flagBytes: bytes,
		flagLines: lines,
		flagWords: words,
		flagChars: chars,
		flagNone:  !bytes && !lines && !words && !chars,

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

// byteCount gets the number of bytes in the currently open file
func (ctx *WCContext) byteCount() (int64, error) {
	return ctx.currentFd().Seek(0, io.SeekEnd)
}

// lineCount gets the number of lines in the currently open file
func (ctx *WCContext) lineCount() (int32, error) {
	var numlines int32 = 0
	fd := ctx.currentFd()

	_, err := fd.Seek(0, io.SeekStart)
	if err != nil {
		return -1, err
	}

	reader := bufio.NewReader(fd)

	for {
		if _, err := reader.ReadBytes('\n'); err == io.EOF {
			return numlines, nil
		} else if err != nil {
			return -1, errors.New("error while reading file contents")
		} else {
			numlines++
		}
	}
}

// wordCount gets the number of words in the currently open file
func (ctx *WCContext) wordCount() (int32, error) {
	var numwords int32 = 0

	fd := ctx.currentFd()

	_, err := fd.Seek(0, io.SeekStart)
	if err != nil {
		return -1, err
	}

	reader := bufio.NewReader(fd)
	word := false

	for {
		if byt, err := reader.ReadByte(); err == io.EOF {
			if word {
				numwords++
			}
			return numwords, nil
		} else if err != nil {
			return -1, errors.New("error while reading file contents")
		} else {
			if isSpecial(byt) && word {
				numwords++
				word = false
			} else if !isSpecial(byt) && !word {
				word = true
			}
		}
	}
}

func (ctx *WCContext) charCount() (int64, error) {
	var numchars int32 = 0

	fd := ctx.currentFd()

	_, err := fd.Seek(0, io.SeekStart)
	if err != nil {
		return -1, err
	}

	reader := bufio.NewReader(fd)

	for {
		if r, _, err := reader.ReadRune(); err == io.EOF {
			return int64(numchars), nil
		} else if err != nil || r == unicode.ReplacementChar {
			return -1, errors.New("error while reading file contents")
		} else {
			numchars++
		}
	}
}

func (ctx *WCContext) Compute() {
	var err error

	ctx.result.Lock()
	defer ctx.result.Unlock()

	if ctx.flagBytes || ctx.flagNone {
		ctx.result.bytes, err = ctx.byteCount()
	}

	if ctx.flagLines || ctx.flagNone {
		ctx.result.lines, err = ctx.lineCount()
	}

	if ctx.flagWords || ctx.flagNone {
		ctx.result.words, err = ctx.wordCount()
	}

	if ctx.flagChars {
		ctx.result.chars, err = ctx.charCount()
	}

	ctx.result.err = err
}

func (ctx *WCContext) currentFile() string {
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
	// check if valid filename passed
	if len(args) != 1 {
		fmt.Println("invalid files passed")
		return
	}

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
