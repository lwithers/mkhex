package hexconv

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"unicode"
)

// BinToHexHelpMessage is an English text explanation of the hexadecimal
// representation used by this package. Each line starts with a comment
// character, so this message can be placed in line with a stream of hexadecimal
// data.
const BinToHexHelpMessage = `# Rules of conversion from hex to binary:
# • Lines with ‘#’ in column 0 are ignored.
# • Address is completely ignored. Hex values are considered to start after the
#   first vertical bar.
# • Spacing of hex digits is not important. Case insensitive conversion is used.
# • Number of hex digits on a line may be changed to insert/remove values, but
#   must always appear as %02x format.
# • Changing hex digits to the following sequences has effects:
#   · “  ” or “--” — byte will be deleted
#   · “<<” or “>>” — byte will be replaced with value from ASCII
#                    (only works with values from 32–126 inclusive)
# • ASCII is ignored (unless replacement above is used). ASCII values are
#   considered to start after the second vertical bar, and only need to be
#   present if replacement above is used.
#
# Convert back with:
#
#   mkhex -r
#
`

// BinToHex converts raw binary data in the input stream into a hexadecimal
// representation written to the output stream.
func BinToHex(inRaw io.Reader, outRaw io.Writer) error {
	var (
		in   = bufio.NewReader(inRaw)
		out  = bufio.NewWriter(outRaw)
		buf  = bytes.NewBuffer(nil)
		line = make([]byte, 16)
		pos  int
		eof  bool
	)
	defer out.Flush()

	for !eof {
		n, err := in.Read(line)
		switch err {
		case io.EOF:
			eof = true
		case nil:
		default:
			return err
		}
		if n < 16 {
			eof = true
		}
		binaryToHexLine(pos, line[:n], buf)
		pos += 16

		if _, err = out.Write(buf.Bytes()); err != nil {
			return err
		}
		buf.Reset()
	}
	return nil
}

var hexdig []byte = []byte("0123456789ABCDEF")

func binaryToHexLine(pos int, line []byte, buf *bytes.Buffer) {
	fmt.Fprintf(buf, "%08X |  ", pos)

	var (
		i int
		b byte
	)
	for i, b = range line {
		buf.WriteByte(hexdig[b>>4])
		buf.WriteByte(hexdig[b&15])
		buf.WriteByte(' ')
		if i == 7 {
			buf.WriteByte(' ')
		}
	}
	for i++; i < 16; i++ {
		buf.WriteString("   ")
		if i == 7 {
			buf.WriteByte(' ')
		}
	}

	buf.WriteString(" | ")
	for _, b = range line {
		if b < 32 || b > 126 {
			buf.WriteRune(unicode.ReplacementChar)
		} else {
			buf.WriteByte(b)
		}
	}
	buf.WriteByte('\n')
}
