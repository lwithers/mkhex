package cmd

import (
	"errors"
	"io"
	"os"

	"github.com/lwithers/mkhex/hexconv"
	"github.com/lwithers/pkg/stdinprompt"
	"github.com/lwithers/pkg/writefile"
	"github.com/spf13/cobra"
)

var binCmd = &cobra.Command{
	Use:   "bin",
	RunE:  binRun,
	Short: "convert a representation to a binary file",
	Long:  `Converts an (ASCII) hex representation of a file back to binary.`,
	Example: `	mkhex bin hex.txt raw.dat
		Converts input file ‘hex.txt’ writing binary into ‘raw.dat’.

	mkhex hex hex.txt
		Converts input file ‘hex.txt’ writing binary to stdout.

	mkhex hex - raw.dat
		Converts hexadecimal presented on stdin, writing binary out to
		‘raw.dat’`,
}

func init() {
	rootCmd.AddCommand(binCmd)
}

func binRun(c *cobra.Command, args []string) error {
	var (
		in       io.Reader = os.Stdin
		out      io.Writer = os.Stdout
		fout     *os.File
		foutName string
		err      error
	)

	switch len(args) {
	case 0:
		// heuristic: if we haven't seen anything after a short delay,
		// prompt the user that we're listening on stdin
		in = stdinprompt.New()
		return hexconv.HexToBin(in, out)

	case 2:
		if args[1] != "-" {
			foutName = args[1]
			foutName, fout, err = writefile.New(foutName)
			if err != nil {
				FatalError(err)
			}
			out = fout
		}

		fallthrough
	case 1:
		if args[0] != "-" {
			if in, err = os.Open(args[0]); err != nil {
				FatalError(err)
			}
		}

	default:
		return errors.New("too many arguments")
	}

	if err = hexconv.HexToBin(in, out); err != nil {
		FatalError(err)
	}

	if fout != nil {
		if err = writefile.Commit(foutName, fout); err != nil {
			FatalError(err)
		}
	}

	return nil
}
