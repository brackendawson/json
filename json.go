// Walking a mile in encoding/json's shoes. I'm doing this to gain a new
// appreciation for and understanding of encoding/json and to learn about
// decoding JSON. I also want to decode json without reading the entire blob
// into a buffer up front. Needless to say you should not use this package.
package json

import (
	"bufio"
	"io"
	"reflect"
	"strconv"
)

var (
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
	boolMap = map[byte]bool{
		't': true,
		'f': false,
	}
	endOf = map[byte][]byte{
		't': []byte(`rue`),
		'f': []byte(`alse`),
		'n': []byte(`ull`),
	}
)

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
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Ptr || vv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	c, err := d.readByte()
	if err != nil {
		return err
	}
	return d.readValue(c, vv)
}

func (d *Decoder) readValue(c byte, v reflect.Value) error {
	var err error

	for {
		switch c {
		case '{':
			return d.readObject(c, v)
		case '[':
			return d.readArray(c, v)
		case '"':
			return d.readString(v)
		case 't', 'f':
			return d.readBool(c, v)
		case 'n':
			return d.readNull()
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return d.readUint(c, v)
		case '-':
			return d.readInt(v)
		case ' ', '\t', '\r', '\n':
		default:
			return d.syntaxErrorf("invalid character %q looking for beginning of value", c)
		}
		if c, err = d.readByte(); err != nil {
			return err
		}
	}
}

func (d *Decoder) readObject(c byte, v reflect.Value) error {
	var (
		obj, val reflect.Value
		key      string
		err      error
		firstKey = true
	)
	switch v.Elem().Kind() {
	case reflect.Interface:
		obj = reflect.ValueOf(&map[string]interface{}{})
	}

objLoop:
	for {
		switch c {
		case ',', '{':
			if c, err = d.readByte(); err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
			if firstKey && c == '}' {
				break objLoop
			}
			firstKey = false

			if key, err = d.readObjectKey(c); err != nil {
				return err
			}

			if err = d.readObjectSeparator(); err != nil {
				return err
			}

			val = reflect.ValueOf(new(interface{}))
			if c, err = d.readByte(); err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
			if err = d.readValue(c, val); err != nil {
				return err
			}

			obj.Elem().SetMapIndex(reflect.ValueOf(key), val.Elem())

			fallthrough
		case ' ', '\t', '\r', '\n':
			if c, err = d.readByte(); err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
		case '}':
			break objLoop
		default:
			return d.syntaxErrorf("invalid character %q after object key:value pair", c)
		}
	}

	v.Elem().Set(obj.Elem())
	return nil
}

func (d *Decoder) readObjectKey(c byte) (string, error) {
	var (
		key string
		err error
	)

keyLoop:
	for {
		switch c {
		case '"':
			if err = d.readString(reflect.ValueOf(&key)); err != nil {
				return "", err
			}
			break keyLoop
		case ' ', '\t', '\r', '\n':
			if c, err = d.readByte(); err != nil {
				return "", err
			}
		default:
			return "", d.syntaxErrorf("invalid character %q looking for beginning of object key string", c)
		}
	}

	return key, nil
}

func (d *Decoder) readObjectSeparator() error {
	var (
		c   byte
		err error
	)
separatorLoop:
	for {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		switch c {
		case ':':
			break separatorLoop
		case ' ', '\t', '\r', '\n':
		default:
			return d.syntaxErrorf("invalid character %q after object key", c)
		}
	}
	return nil
}

func (d *Decoder) readArray(c byte, v reflect.Value) error {
	var (
		i         = 0
		arr, elem reflect.Value
		err       error
		firstElem = true
	)

	switch v.Elem().Kind() {
	case reflect.Interface:
		arr = reflect.ValueOf(&[]interface{}{})
	case reflect.Slice, reflect.Array:
		arr = v
	default:
		return d.unmarshalTypeError("array", v.Elem().Type())
	}

arrLoop:
	for {
		switch c {
		case ',', '[':
			if c, err = d.readByte(); err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
			if firstElem && c == ']' {
				break arrLoop
			}
			firstElem = false

			if i >= arr.Elem().Len() {
				if arr.Elem().Kind() == reflect.Slice {
					arr.Elem().Set(reflect.Append(arr.Elem(), reflect.New(arr.Elem().Type().Elem()).Elem()))
					elem = arr.Elem().Index(i).Addr()
				} else {
					// The Array v has no more space, but we must read the values to be able to proceed
					elem = reflect.ValueOf(new(interface{}))
				}
			} else {
				elem = arr.Elem().Index(i).Addr()
			}
			if err = d.readValue(c, elem); err != nil {
				return err
			}
			i++

			fallthrough
		case ' ', '\t', '\r', '\n':
			if c, err = d.readByte(); err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
		case ']':
			break arrLoop
		default:
			return d.syntaxErrorf("invalid character %q after array element", c)
		}
	}

	if arr.Elem().Kind() == reflect.Slice {
		arr.Elem().SetLen(i)
	}
	v.Elem().Set(arr.Elem())
	return nil
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
				return io.ErrUnexpectedEOF
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
	for i := range endOf[b] {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if c != endOf[b][i] {
			return d.syntaxErrorf("invalid character %q in literal %v (expecting %q)", c, boolMap[b], endOf[b][i])
		}
	}
	if v.Elem().Kind() != reflect.Bool && v.Elem().Kind() != reflect.Interface {
		return d.unmarshalTypeError("bool", v.Elem().Type())
	}
	v.Elem().Set(reflect.ValueOf(boolMap[b]))
	return nil
}

func (d *Decoder) readNull() error {
	var (
		c   byte
		err error
	)
	for i := range endOf['n'] {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if c != endOf['n'][i] {
			return d.syntaxErrorf("invalid character %q in literal null (expecting %q)", c, endOf['n'][i])
		}
	}
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
		if c == '.' || c == 'e' || c == 'E' {
			return d.readFloat(rawNumber, c, v)
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
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if c == '.' || c == 'e' || c == 'E' {
			if len(rawNumber) == 0 {
				return d.syntaxErrorf("invalid character '.' in numeric literal")
			}
			rawNumber = append([]byte{'-'}, rawNumber...)
			return d.readFloat(rawNumber, c, v)
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

func (d *Decoder) readFloat(b []byte, e byte, v reflect.Value) error {
	var (
		c          byte
		err        error
		num        float64
		expo       = false
		signedExpo = false
	)
	b = append(b, e)
	if e == 'e' || e == 'E' {
		expo = true
	}
floatLoop:
	for {
		if c, err = d.readByte(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch {
		case c == 'e', c == 'E':
			if expo {
				return d.syntaxErrorf("invalid character %q in exponent of numeric literal", c)
			}
			expo = true
		case c == '-', c == '+':
			if signedExpo {
				return d.syntaxErrorf("invalid character %q in exponent of numeric literal", c)
			}
			signedExpo = true
		case c >= '0' && c <= '9':
		default:
			if err = d.unreadByte(); err != nil {
				return err
			}
			break floatLoop
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
