package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
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
		input       []byte
		destination interface{}
	}{
		"string_*interface{}": {[]byte(`"string"`), new(interface{})},
		"string_interface{}":  {[]byte(`"string"`), nil},
		"string_*string":      {[]byte(`"string"`), func() *string { s := ""; return &s }()},
		"string_string":       {[]byte(`"string"`), ""},
		"string_*int":         {[]byte(`"string"`), new(int)},
		"string_int":          {[]byte(`"string"`), 0},

		"bool_*interface{}": {[]byte(`true`), new(interface{})},
		"bool_interface{}":  {[]byte(`true`), nil},
		"bool_*bool":        {[]byte(`true`), new(bool)},
		"bool_bool":         {[]byte(`true`), false},
		"bool_*int":         {[]byte(`true`), new(int)},
		"bool_int":          {[]byte(`true`), 0},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			errJ := json.NewDecoder(bytes.NewBuffer(tt.input)).Decode(tt.destination)
			err := NewDecoder(bytes.NewBuffer(tt.input)).Decode(tt.destination)
			if s, ok := tt.destination.(*string); ok {
				assert.Equal(t, "string", *s)
			}
			if b, ok := tt.destination.(*bool); ok {
				assert.True(t, *b)
			}
			eqaulError(t, errJ, err)
		})
	}
}

// TODO test the invalid UTF8 sequences here to lock in behaviour

func TestDecodeReadError(t *testing.T) {
	tests := map[string]string{
		"fist read":   ``,
		"second read": ` `,
		"read string": `"`,
		"unescape":    `"\`,
		"bool":        `t`,
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
			assert.Equal(t, expected.Offset, err2.Offset)
		} else {
			t.Errorf("Incorrect error type %T, expected *SyntaxError: %s", err, err)
		}
	case *json.InvalidUnmarshalError:
		assert.EqualError(t, err, expected.Error())
		if err2, ok := err.(*InvalidUnmarshalError); ok {
			assert.Equal(t, expected.Type, err2.Type)
		} else {
			t.Errorf("Incorrect error type %T, expected *InvalidUnmarshalError: %s", err, err)
		}
	case *json.UnmarshalTypeError:
		assert.EqualError(t, err, expected.Error())
		if err2, ok := err.(*UnmarshalTypeError); ok {
			assert.Equal(t, expected.Type, err2.Type)
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
