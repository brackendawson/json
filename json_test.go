package json

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/intel-go/fastjson"
	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	tests := map[string][]byte{
		"empty json": []byte(``),
		"invalid":    []byte(`lol`),

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

func eqaulError(t *testing.T, expected, err error) {
	switch expected := expected.(type) {
	case *json.SyntaxError:
		assert.EqualError(t, err, expected.Error())
		if err2, ok := err.(*SyntaxError); ok {
			assert.Equal(t, expected.Offset, err2.Offset)
		} else {
			t.Errorf("Incorrect error type %T, expected *SyntaxError: %s", err, err)
		}
	default:
		assert.Equal(t, expected, err)
		t.Logf("Error types: %T, %T", expected, err)
	}
}

// TODO pass a non-pointer to Decode

// TODO test the invalid UTF8 sequences here to lock in behaviour

func BenchmarkDecodeString(b *testing.B) {
	tests := map[string][]byte{
		"small": []byte(`"the cat sat on the mat"`),
		"large": func() []byte {
			out, err := ioutil.ReadFile("fixtures/romeo_and_juliet.txt")
			if err != nil {
				b.Fatal(err)
			}
			return out
		}(),
	}
	for name, tt := range tests {
		b.Run(name, func(b *testing.B) {
			b.Run("github.com/brackendawson/json", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var v interface{}
					if err := NewDecoder(bytes.NewReader(tt)).Decode(&v); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("encoding/json                ", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var v interface{}
					if err := json.NewDecoder(bytes.NewReader(tt)).Decode(&v); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("github.com/intel-go/fastjson ", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					var v interface{}
					if err := fastjson.NewDecoder(bytes.NewReader(tt)).Decode(&v); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
