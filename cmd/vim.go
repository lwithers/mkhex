package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

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
	var editorError error
	for {
		if err := vimLaunch(c, tmpf.Name(), editorError); err != nil {
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
		editorError = err
	}
}

func vimFindExec(plain string) error {
	_, err := exec.LookPath(plain)
	return err
}

func vimLaunch(c *cobra.Command, hexfile string, editorError error) error {
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

	// build the command
	cmd := exec.Command(editor, hexfile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// display errors to user if necessary
	if editorError != nil {
		var qffile string
		if filepath.Base(editor) == "vim" {
			qffile = vimQuickfix(editorError, hexfile)
		}
		if qffile != "" {
			cmd.Args = cmd.Args[:len(cmd.Args)-1] // strip file
			cmd.Args = append(cmd.Args, "-c")
			cmd.Args = append(cmd.Args, "set errorformat=%f:%l:%c:%m")
			cmd.Args = append(cmd.Args, "-c")
			cmd.Args = append(cmd.Args, ":copen 10")
			cmd.Args = append(cmd.Args, "-c")
			cmd.Args = append(cmd.Args, ":cc1")
			cmd.Args = append(cmd.Args, "-q")
			cmd.Args = append(cmd.Args, qffile)
			defer os.Remove(qffile)
		} else {
			// display to user, prompt
			fmt.Printf("*** Errors in conversion:\n%v\n"+
				"Press Enter to continue.\n", editorError)
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
		}
	}

	// run the command
	return cmd.Run()
}

func vimQuickfix(editorError error, hexfile string) string {
	qftmp, err := ioutil.TempFile("", "mkhex-quickfix")
	if err != nil {
		return ""
	}

	if cc, ok := editorError.(hexconv.ConvErrors); ok {
		for _, c := range cc {
			if co, ok := c.(*hexconv.ConvErr); ok {
				fmt.Fprintf(qftmp, "%s:%d:%d:%s\n",
					hexfile, co.Line, co.Col, co.Err)
			} else {
				fmt.Fprintf(qftmp, "%s:1:1:%s\n",
					hexfile, c)
			}
		}
	} else {
		fmt.Fprintf(qftmp, "%s:1:1:%s\n", hexfile, editorError)
	}

	if err = qftmp.Close(); err != nil {
		os.Remove(qftmp.Name())
		return ""
	}
	return qftmp.Name()
}
