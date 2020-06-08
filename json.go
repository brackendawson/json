// Walking a mile in encoding/json's shoes. I'm doing this to gain a new
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
	"strconv"
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
	boolMap = map[byte]bool{
		't': true,
		'f': false,
	}
	boolEnd = map[byte][]byte{
		't': []byte(`rue`),
		'f': []byte(`alse`),
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

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (i *InvalidUnmarshalError) Error() string {
	if i.Type == nil {
		return "json: Unmarshal(nil)"
	}
	return "json: Unmarshal(non-pointer " + i.Type.String() + ")"
}

type UnmarshalTypeError struct {
	Value  string
	Type   reflect.Type
	Offset int64
	Struct string
	Field  string
}

func (u *UnmarshalTypeError) Error() string {
	return "json: cannot unmarshal " + u.Value + " into Go value of type " + u.Type.String()
}

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
		vv  = reflect.ValueOf(v)
		c   byte
		err error
	)
	if vv.Kind() != reflect.Ptr || vv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}
	for {
		c, err = d.readByte()
		switch {
		case err != nil:
			return err
		case c == '"':
			return d.readString(vv)
		case c == 't', c == 'f':
			return d.readBool(c, vv)
		case c >= '0' && c <= '9':
			return d.readUint(c, vv)
		case c == '-':
			return d.readInt(vv)
		case whitespace[c]:
		default:
			return d.syntaxErrorf("invalid character %q looking for beginning of value", c)
		}
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
		case c == '"':
			if v.Elem().Kind() != reflect.String && v.Elem().Kind() != reflect.Interface {
				return d.unmarshalTypeError("string", v.Elem().Type())
			}
			v.Elem().Set(reflect.ValueOf(string(buf)))
			return nil
		case c == '\\':
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

func (d *Decoder) readBool(b byte, v reflect.Value) error {
	var (
		c   byte
		err error
	)
	for i := 0; i < len(boolEnd[b]); i++ {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				return EOF
			}
			return err
		}
		if c != boolEnd[b][i] {
			return d.syntaxErrorf("invalid character %q in literal %v (expecting %q)", c, boolMap[b], boolEnd[b][i])
		}
	}
	if v.Elem().Kind() != reflect.Bool && v.Elem().Kind() != reflect.Interface {
		return d.unmarshalTypeError("bool", v.Elem().Type())
	}
	v.Elem().Set(reflect.ValueOf(boolMap[b]))
	return nil
}

func (d *Decoder) readUint(b byte, v reflect.Value) error {
	var (
		rawNumber = []byte{b}
		c         byte
		err       error
		num       float64
	)
	for {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if c == '.' {
			return d.readFloat(rawNumber, v)
		}
		if c < '0' || c > '9' {
			if err = d.unreadByte(); err != nil {
				return err
			}
			break
		}
		// Number must be minimally encoded
		if rawNumber[0] == '0' {
			break
		}
		rawNumber = append(rawNumber, c)
	}
	num, _ = strconv.ParseFloat(string(rawNumber), 64)
	switch v.Elem().Kind() {
	case reflect.Interface:
		v.Elem().Set(reflect.ValueOf(num))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.Elem().SetUint(uint64(num))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.Elem().SetInt(int64(num))
	case reflect.Float32, reflect.Float64:
		v.Elem().SetFloat(num)
	default:
		return d.unmarshalTypeError("number", v.Elem().Type())
	}
	return nil
}

func (d *Decoder) readInt(v reflect.Value) error {
	var (
		rawNumber []byte
		c         byte
		err       error
		num       float64
		expectEOF = false
	)
	for {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				if expectEOF {
					break
				}
				return EOF
			}
			return err
		}
		if c == '.' {
			if len(rawNumber) == 0 {
				return d.syntaxErrorf("invalid character '.' in numeric literal")
			}
			rawNumber = append([]byte{'-'}, rawNumber...)
			return d.readFloat(rawNumber, v)
		}
		if c < '0' || c > '9' {
			if len(rawNumber) == 0 {
				return d.syntaxErrorf("invalid character %q in numeric literal", c)
			}
			if err = d.unreadByte(); err != nil {
				return err
			}
			break
		}
		// Number must be minimally encoded
		if len(rawNumber) > 0 && rawNumber[0] == '0' {
			break
		}
		rawNumber = append(rawNumber, c)
		expectEOF = true
	}
	num, _ = strconv.ParseFloat("-"+string(rawNumber), 64)
	switch v.Elem().Kind() {
	case reflect.Interface:
		v.Elem().Set(reflect.ValueOf(num))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return d.unmarshalTypeError("number -"+string(rawNumber), v.Elem().Type())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.Elem().SetInt(int64(num))
	case reflect.Float32, reflect.Float64:
		v.Elem().SetFloat(num)
	default:
		return d.unmarshalTypeError("number", v.Elem().Type())
	}
	return nil
}

func (d *Decoder) readFloat(b []byte, v reflect.Value) error {
	var (
		c   byte
		err error
		num float64
	)
	b = append(b, '.')
	for {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if c < '0' || c > '9' {
			break
		}
		b = append(b, c)
	}
	num, _ = strconv.ParseFloat(string(b), 64)
	switch v.Elem().Kind() {
	case reflect.Interface:
		v.Elem().Set(reflect.ValueOf(num))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return d.unmarshalTypeError("number "+string(b), v.Elem().Type())
	case reflect.Float32, reflect.Float64:
		v.Elem().SetFloat(num)
	default:
		return d.unmarshalTypeError("number", v.Elem().Type())
	}
	return nil
}

func (d *Decoder) readByte() (byte, error) {
	c, err := d.in.ReadByte()
	if err != nil {
		return 0, err
	}
	d.offset++
	return c, nil
}

func (d *Decoder) unreadByte() error {
	if err := d.in.UnreadByte(); err != nil {
		return err
	}
	d.offset--
	return nil
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

func (d *Decoder) syntaxErrorf(format string, a ...interface{}) *SyntaxError {
	return &SyntaxError{
		msg:    fmt.Sprintf(format, a...),
		Offset: d.offset,
	}
}

func (d *Decoder) unmarshalTypeError(value string, t reflect.Type) *UnmarshalTypeError {
	return &UnmarshalTypeError{
		Value:  value,
		Type:   t,
		Offset: d.offset,
	}
}
