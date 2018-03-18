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

var hexCmd = &cobra.Command{
	Use:   "hex",
	RunE:  hexRun,
	Short: "convert a file to hex representation",
	Long: `Converts an arbitrary binary file to (ASCII) hex representation. The
hex will be marked up with a header explaining the format, making it amenable to
editing and conversion back to binary using the ‘bin’ command.`,
	Example: `	mkhex hex raw.dat hex.txt
		Converts input file ‘raw.dat’ writing hex into ‘hex.txt’.

	mkhex hex raw.dat
		Converts input file ‘raw.dat’ writing to stdout.

	mkhex hex - hex.txt
		Converts stdin writing to ‘hex.txt’`,
}

func init() {
	rootCmd.AddCommand(hexCmd)
}

func hexRun(c *cobra.Command, args []string) error {
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
		return hexconv.BinToHex(in, out)

	case 2:
		if args[1] != "-" {
			foutName, fout, err = writefile.New(args[1])
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

	if err = hexconv.BinToHex(in, out); err != nil {
		FatalError(err)
	}

	if fout != nil {
		if err = writefile.Commit(foutName, fout); err != nil {
			FatalError(err)
		}
	}

	return nil
}
