package hexconv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// TODO: error type representing multiple errors is required

// ConvErr is returned if a conversion error occurs. This can only occur when
// converting from the textual hexadecimal representation back to binary.
type ConvErr struct {
	// Line at which error was encountered (starting from 1).
	Line int
	// Column at which error was encountered (starting from 1).
	Col int

	// Err is the underlying error.
	Err error
}

// Error returns a string describing the error.
func (c *ConvErr) Error() string {
	return fmt.Sprintf("%d:%d: %s", c.Line, c.Col, c.Err)
}

// HexToBin converts a textual, hexadecimal representation of data into a raw
// binary stream.
func HexToBin(in io.Reader, out io.Writer) error {
	conv, io := hexToBinaryAux(in, out)
	if io != nil {
		return io
	}
	if len(conv) > 0 {
		cstr := make([]string, 0, len(conv))
		for _, c := range conv {
			cstr = append(cstr, c.Error())
		}
		return errors.New(strings.Join(cstr, "\n"))
	}
	return nil
}

func hexToBinaryAux(in io.Reader, out io.Writer) (conv []error, io error) {
	var (
		err     error
		b       = make([]byte, 0, 16)
		scanner = bufio.NewScanner(in)
		line    int
		bout    = bufio.NewWriter(out)
	)
	defer bout.Flush()

	for scanner.Scan() {
		line++
		b, err = hexToBinaryLine(scanner.Bytes(), b[:0])
		if err != nil {
			if cerr, ok := err.(*ConvErr); ok {
				cerr.Line = line
			}
			conv = append(conv, err)
			if len(conv) >= 10 {
				return
			}
		} else if _, err = bout.Write(b); err != nil {
			io = err
			return
		}
	}
	io = scanner.Err()
	return
}

func hexToBinaryLine(in []byte, buf []byte) (out []byte, err error) {
	out = buf
	if len(in) == 0 || in[0] == '#' {
		return
	}

	var ascii []byte
	segments := bytes.SplitN(in, []byte{'|'}, 3)
	if len(segments) < 2 {
		err = &ConvErr{
			Col: len(in),
			Err: errors.New("missing address/data separator"),
		}
		return
	}
	// segments[0] is address, and is ignored
	hexes := bytes.Fields(segments[1])
	if len(segments) == 3 {
		ascii = segments[2]
	}
	// possibly strip a single leading space
	if len(ascii) > 0 && ascii[0] == ' ' {
		ascii = ascii[1:]
	}

	for i, hex := range hexes {
		b, ok := hexToBinaryOne(hex)
		if ok {
			out = append(out, b)
			continue
		}

		if hexToBinaryWantsAscii(hex) {
			if i >= len(ascii) {
				err = &ConvErr{
					Col: len(in),
					Err: fmt.Errorf("ASCII byte %d "+
						"not present", i),
				}
				return
			}
			out = append(out, ascii[i])
			continue
		}

		err = &ConvErr{
			Col: bytes.Index(in, hex),
			Err: fmt.Errorf("invalid hex sequence %q, "+
				"must be 00–FF", hex),
		}
		return
	}
	return
}

func hexToBinaryOne(hex []byte) (b byte, ok bool) {
	if len(hex) != 2 {
		return 0, false
	}
	n, err := strconv.ParseUint(string(hex), 16, 8)
	if err != nil {
		return 0, false
	}
	return byte(n), true
}

func hexToBinaryWantsAscii(hex []byte) bool {
	if len(hex) != 2 {
		return false
	}
	return (hex[0] == '<' && hex[1] == '<') ||
		(hex[0] == '>' && hex[1] == '>')
}
