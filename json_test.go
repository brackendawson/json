package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/intel-go/fastjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDecode(t *testing.T) {
	tests := map[string][]byte{
		"empty json": []byte(``),
		"invalid":    []byte(`lol`),

		"true":          []byte(`true`),
		"false":         []byte(`false`),
		"invalid true":  []byte(`ture`),
		"invalid false": []byte(`fsale`),
		"short true":    []byte(`tru`),
		"short false":   []byte(`fals`),
		"shorter true":  []byte(`t`),
		"shorter false": []byte(`f`),

		"unterm empty string": []byte(`"`),
		"unterm string":       []byte(`" `),
		"empty string":        []byte(`""`),
		"small string":        []byte(`" "`),
		"string":              []byte(`"string"`),
		"longer string":       []byte(`"longer string`),
		"emoji string":        []byte(`"🚀"`),
		"more emoji string":   []byte(`"I 👏 love 👏 emoji 👏"`),
		"multiline string":    []byte("\"not\nallowed\""),
		"windows string":      []byte("\"not\r\nallowed\""),
		"backspace string":    []byte("\"oops\b\b\b\b\""),
		"formfeed string":     []byte("\"what even is a form feed?\f\""),
		"tab string":          []byte("\"tabs\tbreak\tit\""),
		"esc valids string":   []byte(`"newline \n return \r backspace \b formfeed \f tab \t backslash \\ quote \""`),
		"empty esc string":    []byte(`"(for offset)\"`),
		"invalid esc string":  []byte(`"(for an offset)\a(padding)"`),
		// encoding/json handes invalid UTF8 ungracefully https://github.com/golang/go/issues/16282
		// "invalid utf8 2/2 string": []byte("\"\xc3\x28\""),
		// "invalid utf8 2/3 string": []byte("\"\xe2\x28\xa1\""),
		// "invalid utf8 3/3 string": []byte("\"\xe2\x82\x28\""),
		// "invalid utf8 2/4 string": []byte("\"\xf0\x28\x8c\xbc\""),
		// "invalid utf8 3/4 string": []byte("\"\xf0\x90\x28\xbc\""),
		// "invalid utf8 4/4 string": []byte("\"\xf0\x28\x8c\x28\""),
		"whitespace string":       []byte(" \t\r\n \"string with whitespace\" \t\r\n "),
		"two strings":             []byte(`"cant have""two strings"`),
		"spaced strings":          []byte(`   "cant have"   "two strings"   `),
		"trailing invalid string": []byte(`"duck duck" goose`),

		"number 0":                              []byte(`0`),
		"number 1":                              []byte(`1`),
		"number 42":                             []byte(`42`),
		"number 32768":                          []byte(`32768`),
		"number MaxUint64":                      []byte(strconv.FormatUint(math.MaxUint64, 10)),
		"number -1":                             []byte(`-1`),
		"number -666":                           []byte(`-666`),
		"number MinInt64":                       []byte(strconv.FormatInt(math.MinInt64, 10)),
		"number -0":                             []byte(`-0`),
		"number 0.1":                            []byte(`0.1`),
		"number 3.141592654":                    []byte(`3.141592654`),
		"number 1000.1":                         []byte(`1000.1`),
		"number -0.1":                           []byte(`-0.1`),
		"number -999.999":                       []byte(`-999.999`),
		"number SmallestNonZeroFloat64":         []byte(strconv.FormatFloat(math.SmallestNonzeroFloat64, 'f', -1, 64)),
		"number MaxFloat64":                     []byte(strconv.FormatFloat(math.MaxFloat64, 'f', -1, 64)),
		"number SmallestNonZeroNegativeFloat64": []byte(strconv.FormatFloat(-math.SmallestNonzeroFloat64, 'f', -1, 64)),
		"number MinFloat64":                     []byte(strconv.FormatFloat(-math.MaxFloat64, 'f', -1, 64)),
		"number .0":                             []byte(`.0`),
		"number .1":                             []byte(`.1`),
		"number -.1":                            []byte(`-.1`),
		"number 01":                             []byte(`01`),
		"number 001":                            []byte(`001`),
		"number -01":                            []byte(`-01`),
		"number -001":                           []byte(`-001`),
		"number 1.2.3":                          []byte(`1.2.3`),
		"number 1.2.3.4":                        []byte(`1.2.3.4`),
		"number -1.2.3":                         []byte(`-1.2.3`),
		"number -1.2.3.4":                       []byte(`-1.2.3.4`),
		"number -":                              []byte(`-`),
		"number --1":                            []byte(`--1`),
		"number -1-":                            []byte(`-1-`),
		"number -1-2":                           []byte(`-1-2`),
		"number 1-2":                            []byte(`1-2`),
		"number -a":                             []byte(`-a`),
		"number 0a":                             []byte(`0a`),
		"number -0a":                            []byte(`-0a`),
		"number 5345j345":                       []byte(`5345j345`),
		"number -5345j345":                      []byte(`-5345j345`),
		"number 5.345j345":                      []byte(`5.345j345`),
		"number -5.345j345":                     []byte(`-5.345j345`),
		"number 0x1":                            []byte(`0x1`),
		"number 1e6":                            []byte(`1e6`),
		"number 1.1e6":                          []byte(`1.1e6`),
		"number 1E6":                            []byte(`1E6`),
		"number 1.1E6":                          []byte(`1.1E6`),
		"number 1e-6":                           []byte(`1e-6`),
		"number 1.1e-6":                         []byte(`1.1e-6`),
		"number 1E-6":                           []byte(`1E-6`),
		"number 1.1E-6":                         []byte(`1.1E-6`),
		"number 1e+6":                           []byte(`1e+6`),
		"number 1.1e+6":                         []byte(`1.1e+6`),
		"number 1E+6":                           []byte(`1E+6`),
		"number 1.1E+6":                         []byte(`1.1E+6`),
		"number -1e6":                           []byte(`-1e6`),
		"number -1.1e6":                         []byte(`-1.1e6`),
		"number -1E6":                           []byte(`-1E6`),
		"number -1.1E6":                         []byte(`-1.1E6`),
		"number -1e-6":                          []byte(`-1e-6`),
		"number -1.1e-6":                        []byte(`-1.1e-6`),
		"number -1E-6":                          []byte(`-1E-6`),
		"number -1.1E-6":                        []byte(`-1.1E-6`),
		"number -1e+6":                          []byte(`-1e+6`),
		"number -1.1e+6":                        []byte(`-1.1e+6`),
		"number -1E+6":                          []byte(`-1E+6`),
		"number -1.1E+6":                        []byte(`-1.1E+6`),
		"number 1ee6":                           []byte(`1ee6`),
		"number 1eE6":                           []byte(`1eE6`),
		"number 1Ee6":                           []byte(`1Ee6`),
		"number 1EE6":                           []byte(`1EE6`),
		"number 1e--6":                          []byte(`1e--6`),
		"number 1e-+6":                          []byte(`1e-+6`),
		"number 1e+-6":                          []byte(`1e+-6`),
		"number 1e++6":                          []byte(`1e++6`),
		"number 1e6j7":                          []byte(`1e6j7`),
		"number 0e6":                            []byte(`0e6`),
		"number -1ee6":                          []byte(`-1ee6`),
		"number -1e--6":                         []byte(`-1e--6`),
		"number -1.1ee6":                        []byte(`-1.1ee6`),
		"number -1.1e--6":                       []byte(`-1.1e--6`),
		"number 1.1ee6":                         []byte(`1.1ee6`),
		"number 1.1e--6":                        []byte(`1.1e--6`),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Log("Input: ", string(input))
			t.Log("Raw: ", input)
			var data, dataJ interface{}
			errJ := json.NewDecoder(bytes.NewReader(input)).Decode(&dataJ)
			err := NewDecoder(bytes.NewReader(input)).Decode(&data)
			if !assert.Equal(t, dataJ, data) {
				t.Logf("Data types: %T, %T", dataJ, data)
				if strJ, ok := dataJ.(string); ok {
					if str, ok := data.(string); ok {
						t.Logf("Raw strings: %v, %v", []byte(strJ), []byte(str))
					}
				}
			}
			eqaulError(t, errJ, err)
		})
	}
}

func TestDecodeToTypes(t *testing.T) {
	tests := map[string]struct {
		input []byte
		dest  interface{}
		check func(t *testing.T, dest interface{})
	}{
		"string_*interface{}": {[]byte(`"string"`), new(interface{}), nil},
		"string_interface{}":  {[]byte(`"string"`), nil, nil},
		"string_*string": {
			[]byte(`"string"`),
			new(string),
			func(t *testing.T, dest interface{}) {
				s, ok := dest.(*string)
				if assert.True(t, ok) {
					assert.Equal(t, "string", *s)
				}
			},
		},
		"string_string": {[]byte(`"string"`), "", nil},
		"string_*int":   {[]byte(`"string"`), new(int), nil},
		"string_int":    {[]byte(`"string"`), 0, nil},

		"bool_*interface{}": {[]byte(`true`), new(interface{}), nil},
		"bool_interface{}":  {[]byte(`true`), nil, nil},
		"bool_*bool": {
			[]byte(`true`),
			new(bool),
			func(t *testing.T, dest interface{}) {
				b, ok := dest.(*bool)
				if assert.True(t, ok) {
					assert.True(t, *b)
				}
			},
		},
		"bool_bool": {[]byte(`true`), false, nil},
		"bool_*int": {[]byte(`true`), new(int), nil},
		"bool_int":  {[]byte(`true`), 0, nil},

		"uint_*interface{}": {[]byte(`1`), new(interface{}), nil},
		"uint_interface{}":  {[]byte(`1`), nil, nil},
		"uint_*uint64":      {[]byte(`1`), new(uint64), nil},
		"uint_uint64":       {[]byte(`1`), uint64(0), nil},
		"uint_*uint32":      {[]byte(`1`), new(uint32), nil},
		"uint_uint32":       {[]byte(`1`), uint32(0), nil},
		"uint_*uint16":      {[]byte(`1`), new(uint16), nil},
		"uint_uint16":       {[]byte(`1`), uint16(0), nil},
		"uint_*uint8":       {[]byte(`1`), new(uint8), nil},
		"uint_uint8":        {[]byte(`1`), uint8(0), nil},
		"uint_*uint": {
			[]byte(`1`),
			new(uint),
			func(t *testing.T, dest interface{}) {
				u, ok := dest.(*uint)
				if assert.True(t, ok) {
					assert.Equal(t, uint(1), *u)
				}
			},
		},
		"uint_uint":     {[]byte(`1`), uint(0), nil},
		"uint_*int64":   {[]byte(`1`), new(int64), nil},
		"uint_int64":    {[]byte(`1`), int64(0), nil},
		"uint_*int32":   {[]byte(`1`), new(int32), nil},
		"uint_int32":    {[]byte(`1`), int32(0), nil},
		"uint_*int16":   {[]byte(`1`), new(int16), nil},
		"uint_int16":    {[]byte(`1`), int16(0), nil},
		"uint_*int8":    {[]byte(`1`), new(int8), nil},
		"uint_int8":     {[]byte(`1`), int8(0), nil},
		"uint_*int":     {[]byte(`1`), new(int), nil},
		"uint_int":      {[]byte(`1`), int(0), nil},
		"uint_*float64": {[]byte(`1`), new(float64), nil},
		"uint_float64":  {[]byte(`1`), float64(0), nil},
		"uint_*float32": {[]byte(`1`), new(float32), nil},
		"uint_float32":  {[]byte(`1`), float32(0), nil},
		"uint_*string":  {[]byte(`1`), new(string), nil},
		"uint_string":   {[]byte(`1`), "", nil},

		"int_*interface{}": {[]byte(`-1`), new(interface{}), nil},
		"int_interface{}":  {[]byte(`-1`), nil, nil},
		"int_*uint64":      {[]byte(`-1`), new(uint64), nil},
		"int_uint64":       {[]byte(`-1`), uint64(0), nil},
		"int_*uint32":      {[]byte(`-1`), new(uint32), nil},
		"int_uint32":       {[]byte(`-1`), uint32(0), nil},
		"int_*uint16":      {[]byte(`-1`), new(uint16), nil},
		"int_uint16":       {[]byte(`-1`), uint16(0), nil},
		"int_*uint8":       {[]byte(`-1`), new(uint8), nil},
		"int_uint8":        {[]byte(`-1`), uint8(0), nil},
		"int_*uint":        {[]byte(`-1`), new(uint), nil},
		"int_uint":         {[]byte(`-1`), uint(0), nil},
		"int_*int64":       {[]byte(`-1`), new(int64), nil},
		"int_int64":        {[]byte(`-1`), int64(0), nil},
		"int_*int32":       {[]byte(`-1`), new(int32), nil},
		"int_int32":        {[]byte(`-1`), int32(0), nil},
		"int_*int16":       {[]byte(`-1`), new(int16), nil},
		"int_int16":        {[]byte(`-1`), int16(0), nil},
		"int_*int8":        {[]byte(`-1`), new(int8), nil},
		"int_int8":         {[]byte(`-1`), int8(0), nil},
		"int_*int": {
			[]byte(`-1`),
			new(int),
			func(t *testing.T, dest interface{}) {
				i, ok := dest.(*int)
				if assert.True(t, ok) {
					assert.Equal(t, -1, *i)
				}
			},
		},
		"int_int":      {[]byte(`-1`), int(0), nil},
		"int_*float64": {[]byte(`-1`), new(float64), nil},
		"int_float64":  {[]byte(`-1`), float64(0), nil},
		"int_*float32": {[]byte(`-1`), new(float32), nil},
		"int_float32":  {[]byte(`-1`), float32(0), nil},
		"int_*string":  {[]byte(`-1`), new(string), nil},
		"int_string":   {[]byte(`-1`), "", nil},

		"float_*interface{}": {[]byte(`1.2`), new(interface{}), nil},
		"float_interface{}":  {[]byte(`1.2`), nil, nil},
		"float_*uint64":      {[]byte(`1.2`), new(uint64), nil},
		"float_uint64":       {[]byte(`1.2`), uint64(0), nil},
		"float_*uint32":      {[]byte(`1.2`), new(uint32), nil},
		"float_uint32":       {[]byte(`1.2`), uint32(0), nil},
		"float_*uint16":      {[]byte(`1.2`), new(uint16), nil},
		"float_uint16":       {[]byte(`1.2`), uint16(0), nil},
		"float_*uint8":       {[]byte(`1.2`), new(uint8), nil},
		"float_*uint8_long":  {[]byte(`12.3`), new(uint8), nil},
		"float_*uint8_vlong": {[]byte(`1234567890.2`), new(uint8), nil},
		"float_uint8":        {[]byte(`1.2`), uint8(0), nil},
		"float_*uint":        {[]byte(`1.2`), new(uint), nil},
		"float_uint":         {[]byte(`1.2`), uint(0), nil},
		"float_*int64":       {[]byte(`1.2`), new(int64), nil},
		"float_int64":        {[]byte(`1.2`), int64(0), nil},
		"float_*int32":       {[]byte(`1.2`), new(int32), nil},
		"float_int32":        {[]byte(`1.2`), int32(0), nil},
		"float_*int16":       {[]byte(`1.2`), new(int16), nil},
		"float_int16":        {[]byte(`1.2`), int16(0), nil},
		"float_*int8":        {[]byte(`1.2`), new(int8), nil},
		"float_int8":         {[]byte(`1.2`), int8(0), nil},
		"float_*int":         {[]byte(`1.2`), new(int), nil},
		"float_int":          {[]byte(`1.2`), int(0), nil},
		"float_*float64": {
			[]byte(`1.2`),
			new(float64),
			func(t *testing.T, dest interface{}) {
				f, ok := dest.(*float64)
				if assert.True(t, ok) {
					assert.Equal(t, float64(1.2), *f)
				}
			},
		},
		"float_float64":  {[]byte(`1.2`), float64(0), nil},
		"float_*float32": {[]byte(`1.2`), new(float32), nil},
		"float_float32":  {[]byte(`1.2`), float32(0), nil},
		"float_*string":  {[]byte(`1.2`), new(string), nil},
		"float_string":   {[]byte(`1.2`), "", nil},

		"negfloat_*interface{}": {[]byte(`-1.2`), new(interface{}), nil},
		"negfloat_interface{}":  {[]byte(`-1.2`), nil, nil},
		"negfloat_*uint64":      {[]byte(`-1.2`), new(uint64), nil},
		"negfloat_uint64":       {[]byte(`-1.2`), uint64(0), nil},
		"negfloat_*uint32":      {[]byte(`-1.2`), new(uint32), nil},
		"negfloat_uint32":       {[]byte(`-1.2`), uint32(0), nil},
		"negfloat_*uint16":      {[]byte(`-1.2`), new(uint16), nil},
		"negfloat_uint16":       {[]byte(`-1.2`), uint16(0), nil},
		"negfloat_*uint8":       {[]byte(`-1.2`), new(uint8), nil},
		"negfloat_*uint8_long":  {[]byte(`-12.3`), new(uint8), nil},
		"negfloat_*uint8_vlong": {[]byte(`-1234567890.2`), new(uint8), nil},
		"negfloat_uint8":        {[]byte(`-1.2`), uint8(0), nil},
		"negfloat_*uint":        {[]byte(`-1.2`), new(uint), nil},
		"negfloat_uint":         {[]byte(`-1.2`), uint(0), nil},
		"negfloat_*int64":       {[]byte(`-1.2`), new(int64), nil},
		"negfloat_int64":        {[]byte(`-1.2`), int64(0), nil},
		"negfloat_*int32":       {[]byte(`-1.2`), new(int32), nil},
		"negfloat_int32":        {[]byte(`-1.2`), int32(0), nil},
		"negfloat_*int16":       {[]byte(`-1.2`), new(int16), nil},
		"negfloat_int16":        {[]byte(`-1.2`), int16(0), nil},
		"negfloat_*int8":        {[]byte(`-1.2`), new(int8), nil},
		"negfloat_int8":         {[]byte(`-1.2`), int8(0), nil},
		"negfloat_*int":         {[]byte(`-1.2`), new(int), nil},
		"negfloat_int":          {[]byte(`-1.2`), int(0), nil},
		"negfloat_*float64": {
			[]byte(`-1.2`),
			new(float64),
			func(t *testing.T, dest interface{}) {
				f, ok := dest.(*float64)
				if assert.True(t, ok) {
					assert.Equal(t, float64(-1.2), *f)
				}
			},
		},
		"negfloat_float64":  {[]byte(`-1.2`), float64(0), nil},
		"negfloat_*float32": {[]byte(`-1.2`), new(float32), nil},
		"negfloat_float32":  {[]byte(`-1.2`), float32(0), nil},
		"negfloat_*string":  {[]byte(`-1.2`), new(string), nil},
		"negfloat_string":   {[]byte(`-1.2`), "", nil},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Log(string(tt.input))
			errJ := json.NewDecoder(bytes.NewBuffer(tt.input)).Decode(tt.dest)
			err := NewDecoder(bytes.NewBuffer(tt.input)).Decode(tt.dest)
			if tt.check != nil {
				tt.check(t, tt.dest)
			}
			eqaulError(t, errJ, err)
		})
	}
}

// TODO test the invalid UTF8 sequences here to lock in behaviour

// TODO decode into *json.RawMessage

func TestDecodeReadError(t *testing.T) {
	tests := map[string]string{
		"fist read":   ``,
		"second read": ` `,
		"read string": `"`,
		"unescape":    `"\`,
		"bool":        `t`,
		"uint":        `0`,
		"uint2":       `10`,
		"int":         `-`,
		"int2":        `-1`,
		"float":       `0.`,
		"float2":      `0.1`,
		"expo":        `0.1e6`,
		"expo2":       `0.1e`,
		"expo3":       `0.1e-`,
		"expo4":       `0.1e-6`,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			primeMock := func(r *mockReader) {
				r.Test(t)
				t.Cleanup(func() { r.AssertExpectations(t) })
				for _, b := range []byte(test) {
					func(b byte) {
						r.On("Read", mock.Anything).Run(func(args mock.Arguments) {
							p := args.Get(0).([]byte)
							require.GreaterOrEqual(t, len(p), 1)
							p[0] = b
						}).Return(1, nil).Once()
					}(b)
				}
				r.On("Read", mock.Anything).Return(0, errors.New("lol")).Once()
			}
			r := &mockReader{}
			var x interface{}
			primeMock(r)
			errJ := json.NewDecoder(r).Decode(&x)
			r = &mockReader{}
			primeMock(r)
			err := NewDecoder(r).Decode(&x)
			eqaulError(t, errJ, err)
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	tests, err := ioutil.ReadDir("fixtures")
	require.NoError(b, err)
	for _, file := range tests {
		b.Run(file.Name(), func(b *testing.B) {
			input, err := ioutil.ReadFile(filepath.Join("fixtures", file.Name()))
			require.NoError(b, err)

			b.Run("github.com/brackendawson/json", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var v interface{}
					if err := NewDecoder(bytes.NewReader(input)).Decode(&v); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("encoding/json                ", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var v interface{}
					if err := json.NewDecoder(bytes.NewReader(input)).Decode(&v); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("github.com/intel-go/fastjson ", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var v interface{}
					if err := fastjson.NewDecoder(bytes.NewReader(input)).Decode(&v); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func eqaulError(t *testing.T, expected, err error) {
	switch expected := expected.(type) {
	case *json.SyntaxError:
		assert.EqualError(t, err, expected.Error())
		if err2, ok := err.(*SyntaxError); ok {
			assert.Equal(t, expected.Offset, err2.Offset, "bad Offset")
		} else {
			t.Errorf("Incorrect error type %T, expected *SyntaxError: %s", err, err)
		}
	case *json.InvalidUnmarshalError:
		assert.EqualError(t, err, expected.Error())
		if err2, ok := err.(*InvalidUnmarshalError); ok {
			assert.Equal(t, expected.Type, err2.Type, "bad Type")
		} else {
			t.Errorf("Incorrect error type %T, expected *InvalidUnmarshalError: %s", err, err)
		}
	case *json.UnmarshalTypeError:
		assert.EqualError(t, err, expected.Error())
		if err2, ok := err.(*UnmarshalTypeError); ok {
			assert.Equal(t, expected.Value, err2.Value, "bad Value")
			assert.Equal(t, expected.Type, err2.Type, "bad Type")
			assert.Equal(t, expected.Offset, err2.Offset, "bad Offset")
			assert.Equal(t, expected.Struct, err2.Struct, "bad Struct")
			assert.Equal(t, expected.Field, err2.Field, "bad Field")
		} else {
			t.Errorf("Incorrect error type %T, expected *UnmarshalTypeError: %s", err, err)
		}
	default:
		assert.Equal(t, expected, err)
		t.Logf("Error types: %T, %T", expected, err)
	}
}

type mockReader struct {
	mock.Mock
}

func (m *mockReader) Read(b []byte) (int, error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}
