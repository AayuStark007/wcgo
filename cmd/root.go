package cmd

import (
	"github.com/aayustark007/wcgo/internal"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "wcgo [file]...",
		Short: "A go implementation of wc to print the newline, char, bytes count for input file(s)",
		Args:  cobra.MinimumNArgs(0),
		Run:   internal.Handle,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&internal.Bytes, "bytes", "c", false, "print the byte counts")
	rootCmd.PersistentFlags().BoolVarP(&internal.Lines, "lines", "l", false, "print the newline counts")
	rootCmd.PersistentFlags().BoolVarP(&internal.Words, "words", "w", false, "print the word counts")
	rootCmd.PersistentFlags().BoolVarP(&internal.Chars, "chars", "m", false, "print the character counts")
}
