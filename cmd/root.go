package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "mkhex",
}

func FatalError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func FatalErrorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra will already have printed the error
		os.Exit(2)
	}
}
