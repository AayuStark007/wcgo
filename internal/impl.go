//go:build linux
// +build linux

package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
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
	words int64
	lines int64
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

	var wg sync.WaitGroup
	defer wg.Wait()

	for idx, file := range ctx.files {
		wg.Add(1)
		go func(i int, f string) {
			defer wg.Done()
			if fd, err := os.Open(f); err != nil {
				ctx.results[i].err = err
			} else {
				ctx.computeInternal(fd, i)
			}
		}(idx, file)
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
	var countLines int64 = 0
	var countWords int64 = 0
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
	var sumLines int64 = 0
	var sumWords int64 = 0
	var sumChars int64 = 0

	var maxWBytes int = 0
	var maxWLines int = 0
	var maxWWords int = 0
	var maxWChars int = 0

	for idx := range ctx.files {
		res := ctx.results[idx]

		if res.err != nil {
			continue
		}
		sumChars += res.chars
		sumLines += res.lines
		sumWords += res.words
		sumBytes += res.bytes

		maxWBytes = maxInt(maxWBytes, intWidth(res.bytes))
		maxWLines = maxInt(maxWLines, intWidth(res.lines))
		maxWWords = maxInt(maxWWords, intWidth(res.words))
		maxWChars = maxInt(maxWChars, intWidth(res.chars))
	}

	maxWBytes = maxInt(maxWBytes, intWidth(sumChars)) + 2
	maxWLines = maxInt(maxWLines, intWidth(sumLines)) + 2
	maxWWords = maxInt(maxWWords, intWidth(sumWords)) + 2
	maxWChars = maxInt(maxWChars, intWidth(sumBytes)) + 2

	// generate for each file
	for idx, file := range ctx.files {
		res := ctx.results[idx]

		if res.err != nil {
			result.WriteString(fmt.Sprintf("%s: %s\n", file, res.err.Error()))
			continue
		}

		if ctx.flagChars {
			result.WriteString(toPaddedStr(maxWChars, res.chars))
		}

		if ctx.flagLines || ctx.flagNone {
			result.WriteString(toPaddedStr(maxWLines, res.lines))
		}

		if ctx.flagWords || ctx.flagNone {
			result.WriteString(toPaddedStr(maxWWords, res.words))
		}

		if ctx.flagBytes || ctx.flagNone {
			result.WriteString(toPaddedStr(maxWBytes, res.bytes))
		}

		fileName := file
		if file == "-" {
			fileName = ""
		}

		result.WriteString(fmt.Sprintf("%s\n", fileName))
	}

	// print the totals
	if len(ctx.files) > 1 {
		if ctx.flagChars {
			result.WriteString(toPaddedStr(maxWChars, sumChars))
		}

		if ctx.flagLines || ctx.flagNone {
			result.WriteString(toPaddedStr(maxWLines, sumLines))
		}

		if ctx.flagWords || ctx.flagNone {
			result.WriteString(toPaddedStr(maxWWords, sumWords))
		}

		if ctx.flagBytes || ctx.flagNone {
			result.WriteString(toPaddedStr(maxWBytes, sumBytes))
		}

		result.WriteString(fmt.Sprintf("%s\n", "total"))
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
	if char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '\f' || char == '\v' {
		return true
	} else {
		return false
	}
}

// lineCount gets the number of lines in the currently open file
func lineCount(data []byte, size int) int64 {
	var count int64 = 0

	for i := 0; i < size; i++ {
		if data[i] == '\n' {
			count++
		}
	}

	return count
}

// wordCount gets the number of words in the currently open file
func wordCount(data []byte, size int, word bool) (int64, bool) {
	var count int64 = 0

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

func intWidth(n int64) int {
	if n == 0 {
		return 1
	}

	if n < 0 {
		n = n * -1
	}

	return int(math.Log10(float64(n))) + 1
}

func maxInt(x, y int) int {
	return int(math.Max(float64(x), float64(y)))
}

func toPaddedStr(width int, value int64) string {
	format := "%" + strconv.Itoa(width) + "s "
	return fmt.Sprintf(format, strconv.FormatInt(value, 10))
}
