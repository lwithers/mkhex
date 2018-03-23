package cmd

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/lwithers/mkhex/hexconv"
	"github.com/lwithers/pkg/writefile"
	"github.com/spf13/cobra"
)

var vimCmd = &cobra.Command{
	Use:   "vim",
	RunE:  vimRun,
	Short: "edit a binary file in vim",
	Long: `Opens a hexadecimal representaton of the file in vim (or
$EDITOR). Upon saving and exiting, the modified hexadecimal will be converted
back to binary and the original file overwritten.`,
	Example: `	mkhex vim raw.dat
		Edits the file ‘raw.dat’.

	mkhex vim --ro raw.dat
		Opens the file ‘raw.dat’ in text editor, but does not cause any
		changes to be saved back to the original.`,
}

var (
	vimRO   bool
	vimExec string
)

func init() {
	rootCmd.AddCommand(vimCmd)
	vimCmd.PersistentFlags().BoolVarP(&vimRO, "ro", "r", false,
		"read-only mode")
	vimCmd.PersistentFlags().StringVarP(&vimExec, "editor", "e", "",
		"editor executable to use")
}

func vimRun(c *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("expected exactly one argument: filename")
	}
	origFname := args[0]

	// perform bin→hex conversion into temporary file
	tmpf, err := ioutil.TempFile("", "mkhex")
	if err != nil {
		FatalErrorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpf.Name())

	fin, err := os.Open(origFname)
	if err != nil {
		FatalError(err)
	}
	if err := hexconv.BinToHex(fin, tmpf); err != nil {
		FatalErrorf("failed to convert %s: %v", origFname, err)
	}
	fin.Close()

	// modify temporary file
	for {
		if err := vimLaunch(c, tmpf.Name()); err != nil {
			FatalErrorf("failed to launch editor: %v", err)
		}

		if vimRO {
			// changes are discarded — return now
			return nil
		}

		// convert back
		fin, err = os.Open(tmpf.Name())
		if err != nil {
			FatalErrorf("failed to open hex file: %v", err)
		}
		foutName, fout, err := writefile.New(origFname)
		if err != nil {
			FatalErrorf("failed to convert hex file: %v", err)
		}

		err = hexconv.HexToBin(fin, fout)
		fin.Close()
		if err == nil {
			// conversion succeeded
			if err = writefile.Commit(foutName, fout); err != nil {
				FatalErrorf("failed to write output: %v", err)
			}
			return nil
		}

		// abort conversion, send back to editor
		writefile.Abort(fout)

		// TODO — notify user of errors
	}
}

func vimFindExec(plain string) error {
	_, err := exec.LookPath(plain)
	return err
}

func vimLaunch(c *cobra.Command, hexfile string) error {
	// find an editor
	var editor string
	switch {
	case vimExec != "":
		editor = vimExec
	case os.Getenv("EDITOR") != "":
		editor = os.Getenv("EDITOR")
	case vimFindExec("vim") == nil:
		editor = "vim"
	case vimFindExec("vi") == nil:
		editor = "vi"
	case vimFindExec("nano") == nil:
		editor = "nano"
	}

	// TODO: if vim, build an error list file; else display errors on stderr
	// and prompt user

	// build the command
	cmd := exec.Command(editor, hexfile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// run the command
	return cmd.Run()
}
