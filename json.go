// Walking a mile in Go's encoding/json's shoes. I'm doing this to gain a new
// appreciation for and understanding of encoding/json and to learn about
// decoding JSON. I also want to decode json without reading the entire blob
// into a buffer up front. Needless to say you should not use this package.
package json

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
)

var (
	EOF      = errors.New("unexpected EOF")
	invalidS = map[byte]bool{
		'\b': true,
		'\f': true,
		'\n': true,
		'\r': true,
		'\t': true,
	}
	escapable = map[byte]byte{
		'b':  '\b',
		'f':  '\f',
		'n':  '\n',
		'r':  '\r',
		't':  '\t',
		'\\': '\\',
		'"':  '"',
	}
	whitespace = map[byte]bool{
		' ':  true,
		'\t': true,
		'\r': true,
		'\n': true,
	}
	TODO = errors.New("TODO")
)

type SyntaxError struct {
	msg    string
	Offset int64
}

func (s *SyntaxError) Error() string {
	return s.msg
}

// TODO Is()?

type Decoder struct {
	in     *bufio.Reader
	offset int64
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		in: bufio.NewReader(r),
	}
}

func (d *Decoder) Decode(v interface{}) error {
	var (
		rv        = reflect.ValueOf(v)
		c         byte
		err       error
		expectEOF = false
	)
	for {
		c, err = d.readByte()
		switch {
		case err != nil:
			if expectEOF && err == io.EOF {
				return nil
			}
			return err
		case c == byte('"'):
			return d.readString(rv)
		case whitespace[c]:
		default:
			return d.syntaxErrorf("invalid character %q looking for beginning of value", c)
		}
		expectEOF = true
	}
}

func (d *Decoder) readString(v reflect.Value) error {
	var (
		buf = []byte{}
		c   byte
		err error
	)
	for {
		c, err = d.readByte()
		switch {
		case err != nil:
			if err == io.EOF {
				return EOF
			}
			return err
		case c == byte('"'):
			v.Elem().Set(reflect.ValueOf(string(buf)))
			return nil
		case c == byte('\\'):
			if c, err = d.unEscape(); err != nil {
				return err
			}
			buf = append(buf, c)
		default:
			if invalidS[c] {
				return d.syntaxErrorf("invalid character %q in string literal", c)
			}
			buf = append(buf, c)
		}
	}
}

func (d *Decoder) syntaxErrorf(format string, a ...interface{}) *SyntaxError {
	return &SyntaxError{
		msg:    fmt.Sprintf(format, a...),
		Offset: d.offset,
	}
}

func (d *Decoder) readByte() (byte, error) {
	d.offset++
	return d.in.ReadByte()
}

func (d *Decoder) unEscape() (byte, error) {
	c, err := d.readByte()
	if err != nil {
		return 0, err
	}
	ec := escapable[c]
	if ec == 0 {
		return 0, d.syntaxErrorf("invalid character %q in string escape code", c)
	}
	return ec, nil
}
