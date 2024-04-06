//go:build linux
// +build linux

package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
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
	flagBytes bool
	flagLines bool
	flagWords bool
	flagChars bool
	flagNone  bool
	flagStdin bool

	files   []string
	results []wcresult
}

type wcresult struct {
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
		files = []string{"-"}
	}

	return &WCContext{
		flagBytes: bytes,
		flagLines: lines,
		flagWords: words,
		flagChars: chars,
		flagNone:  !bytes && !lines && !words && !chars,
		flagStdin: flagStdin,

		files:   files,
		results: make([]wcresult, len(files)),
	}, nil
}

func (ctx *WCContext) Compute() {
	if ctx.flagStdin {
		ctx.computeInternal(os.Stdin, 0)
		return
	}

	for idx, file := range ctx.files {
		if fd, err := os.Open(file); err != nil {
			ctx.results[idx].err = err
		} else {
			ctx.computeInternal(fd, idx)
		}
	}
}

func (ctx *WCContext) computeInternal(fd *os.File, index int) {
	defer fd.Close()

	if !ctx.flagStdin {
		unix.Fadvise(int(fd.Fd()), 0, 0, unix.FADV_SEQUENTIAL)
	}

	reader := bufio.NewReaderSize(fd, CHUNK_SIZE)
	buff := make([]byte, CHUNK_SIZE)
	var countBytes int64 = 0
	var countLines int32 = 0
	var countWords int32 = 0
	var countChars int64 = 0

	word := false

	for {
		numBytes, err := reader.Read(buff)
		if err == io.EOF || numBytes == 0 {
			if word {
				countWords++
			}
			ctx.results[index].bytes = countBytes
			ctx.results[index].lines = countLines
			ctx.results[index].words = countWords
			ctx.results[index].chars = countChars

			return
		}

		countBytes += int64(numBytes)
		countLines += lineCount(buff, numBytes)
		// special handling in case we end buffer in middle of a word
		count, wrd := wordCount(buff, numBytes, word)
		word = wrd
		countWords += count

		if ctx.flagChars {
			countChars += charCount(bytes.NewReader(buff[:numBytes]))
		}
	}
}

func (ctx *WCContext) String() string {
	var result strings.Builder

	var sumBytes int64 = 0
	var sumLines int32 = 0
	var sumWords int32 = 0
	var sumChars int64 = 0

	// generate for each file
	for idx, file := range ctx.files {
		res := ctx.results[idx]

		if res.err != nil {
			result.WriteString(fmt.Sprintf("%s: %s\n", file, res.err.Error()))
			continue
		}

		if ctx.flagChars {
			sumChars += res.chars
			result.WriteString(fmt.Sprintf("  %d", res.chars))
		}

		if ctx.flagLines || ctx.flagNone {
			sumLines += res.lines
			result.WriteString(fmt.Sprintf("  %d", res.lines))
		}

		if ctx.flagWords || ctx.flagNone {
			sumWords += res.words
			result.WriteString(fmt.Sprintf("  %d", res.words))
		}

		if ctx.flagBytes || ctx.flagNone {
			sumBytes += res.bytes
			result.WriteString(fmt.Sprintf("  %d", res.bytes))
		}

		fileName := file
		if file == "-" {
			fileName = ""
		}

		result.WriteString(fmt.Sprintf(" %s\n", fileName))
	}

	if len(ctx.files) > 1 {
		if ctx.flagBytes || ctx.flagNone {
			result.WriteString(fmt.Sprintf("  %d", sumBytes))
		}

		if ctx.flagLines || ctx.flagNone {
			result.WriteString(fmt.Sprintf("  %d", sumLines))
		}

		if ctx.flagWords || ctx.flagNone {
			result.WriteString(fmt.Sprintf("  %d", sumWords))
		}

		if ctx.flagChars {
			result.WriteString(fmt.Sprintf("  %d", sumChars))
		}

		result.WriteString(fmt.Sprintf(" %s\n", "total"))
	}

	return result.String()
}

func Handle(cmd *cobra.Command, args []string) {
	ctx, err := New(args, Bytes, Lines, Words, Chars)
	if err != nil {
		panic(err)
	}

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
func lineCount(data []byte, size int) int32 {
	var count int32 = 0

	for i := 0; i < size; i++ {
		if data[i] == '\n' {
			count++
		}
	}

	return count
}

// wordCount gets the number of words in the currently open file
func wordCount(data []byte, size int, word bool) (int32, bool) {
	var count int32 = 0

	for i := 0; i < size; i++ {
		spl := isSpecial(data[i])
		if spl && word {
			count++
			word = false
		} else if !spl && !word {
			word = true
		}
	}

	return count, word
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
