package internal

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var Bytes bool

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

	if Bytes {
		if offset, err := fd.Seek(0, io.SeekEnd); err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("%d %s\n", offset, fileName)
		}
	}
}
