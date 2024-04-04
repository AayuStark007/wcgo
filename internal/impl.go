package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	Bytes bool
	Lines bool
	Words bool
)

func Handle(cmd *cobra.Command, args []string) {
	// check if valid filename passed
	if len(args) != 1 {
		fmt.Println("invalid files passed")
		return
	}

	fileName := args[0]
	fd, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("error opening file %s\n", err.Error())
		return
	}
	defer fd.Close()

	if Bytes {
		if offset, err := fd.Seek(0, io.SeekEnd); err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("%d %s\n", offset, fileName)
		}
	}

	if Lines {
		numlines := 0
		reader := bufio.NewReader(fd)

		for {
			if _, err := reader.ReadBytes('\n'); err == io.EOF {
				fmt.Printf("%d %s\n", numlines, fileName)
				return
			} else if err != nil {
				fmt.Printf("error reading file %s\n", err.Error())
				return
			} else {
				numlines++
			}
		}
	}

	if Words {
		numwords := 0
		reader := bufio.NewReader(fd)
		word := false
		buf := []byte{}

		for {
			if byt, err := reader.ReadByte(); err == io.EOF {
				if word {
					numwords++
				}
				fmt.Printf("%d %s\n", numwords, fileName)
				return
			} else if err != nil {
				fmt.Printf("error reading file %s\n", err.Error())
				return
			} else {
				if isSpecial(byt) && word {
					numwords++
					word = false
					// fmt.Println(string(buf))
					buf = buf[:0]
				} else if !isSpecial(byt) && !word {
					word = true
				}

				if word {
					buf = append(buf, byt)
				}
			}
		}
	}
}

func isSpecial(char byte) bool {
	if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
		return true
	} else {
		return false
	}
}
