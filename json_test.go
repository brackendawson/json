package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"path/filepath"
	"reflect"
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

		"null":         []byte(`null`),
		"short null":   []byte(`nul`),
		"shorter null": []byte(`n`),
		"wrong null":   []byte(`nil`),
		"trailed null": []byte(`[null,1]`),

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
		"path string":         []byte(`"/usr/local/bin/go"`),
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
		"formfeed space":          []byte("\f\"what even is a form feed?\""),
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

		"empty array":    []byte(`[]`),
		"1 num array":    []byte(`[1]`),
		"2 num array":    []byte(`[1,2]`),
		"3 num array":    []byte(`[-1,0,1]`),
		"1 string array": []byte(`["lol"]`),
		"2 string array": []byte(`["lol","wot"]`),
		"1 bool array":   []byte(`[true]`),
		"2 bool array":   []byte(`[true, false]`),
		"1 float array":  []byte(`[1.1]`),
		"2 float array":  []byte(`[1.1,-2.2]`),
		"mixed array":    []byte(`[42,-7,3.141592654,"hello\nworld\n",true]`),
		"spaced array":   []byte(" \t\n\r [ \t\n\r 42 \t\n\r , \t\n\r -7 \t\n\r ,  3.141592654  ,  \"hello\\nworld\\n\"  ,  true \t\n\r ] \t\n\r "),
		"smnested array": []byte(`[[[1]]]`),
		"nested array":   []byte(`[[1,2],[3,4]]`),
		"very nested array": []byte(`[[[1,2,3],[4,5,6],[7,8,9]],
		[["a","b","c"],["d","e","f"],["g","h","i"]],
			[[true,false,true],[false,true,false],[true,false,true]]]`),
		"unterm array":      []byte(`[`),
		"unterm2 array":     []byte(`["`),
		"unterm3 array":     []byte(`["a`),
		"unterm4 array":     []byte(`["a"`),
		"unterm5 array":     []byte(`["a",`),
		"unterm6 array":     []byte(`["a","`),
		"unterm7 array":     []byte(`["a","b`),
		"unterm8 array":     []byte(`["a","b"`),
		"unexpect array":    []byte(`~["a","b"]`),
		"unexpect2 array":   []byte(`[~"a","b"]`),
		"unexpect3 array":   []byte(`["a"~,"b"]`),
		"unexpect4 array":   []byte(`["a",~"b"]`),
		"unexpect5 array":   []byte(`["a","b"~]`),
		"unexpect6 array":   []byte(`["a","b"]~`),
		"unterm popd array": []byte(`[1`),
		"unterm sepd array": []byte(`[1,`),
		"early termd array": []byte(`[1,]`),
		"unsepd array":      []byte(`[1 2]`),
		"trailed array":     []byte(`[1,2]trail`),
		"doublesepd array":  []byte(`[1,,2]`),
		"valueless array":   []byte(`[,]`),

		"empty object":  []byte(`{}`),
		"simple object": []byte(`{"a":1}`),
		"bigger object": []byte(`{"a":1,"b":2}`),
		"spaced object": []byte(" \t\r\n { \t\r\n \"a\" \t\r\n : \t\r\n 1 \t\r\n , \t\r\n \"b\" \t\r\n : \t\r\n 2 \t\r\n } \t\r\n "),
		"mixed object": []byte(`{
			"string":	"hi",
			"uint":		1,
			"int":		-1,
			"float":	1.1,
			"bool":		true,
			"array":	[],
			"object":	{}
		}`),
		"unterm object":     []byte(`{`),
		"unterm2 object":    []byte(`{"`),
		"unterm3 object":    []byte(`{"a`),
		"unterm4 object":    []byte(`{"a"`),
		"unterm5 object":    []byte(`{"a":`),
		"unterm6 object":    []byte(`{"a":"`),
		"unterm7 object":    []byte(`{"a":"a`),
		"unterm8 object":    []byte(`{"a":"a"`),
		"unterm9 object":    []byte(`{"a":"a",`),
		"unterm10 object":   []byte(`{"a":"a","`),
		"unterm11 object":   []byte(`{"a":"a","b`),
		"unterm12 object":   []byte(`{"a":"a","b"`),
		"unterm13 object":   []byte(`{"a":"a","b":`),
		"unterm14 object":   []byte(`{"a":"a","b":"`),
		"unterm15 object":   []byte(`{"a":"a","b":"b`),
		"unterm16 object":   []byte(`{"a":"a","b":"b"`),
		"unexpect object":   []byte(`~{"a":"a","b":"b"}`),
		"unexpect2 object":  []byte(`{~"a":"a","b":"b"}`),
		"unexpect3 object":  []byte(`{"a"~:"a","b":"b"}`),
		"unexpect4 object":  []byte(`{"a":~"a","b":"b"}`),
		"unexpect5 object":  []byte(`{"a":"a"~,"b":"b"}`),
		"unexpect6 object":  []byte(`{"a":"a",~"b":"b"}`),
		"unexpect7 object":  []byte(`{"a":"a","b"~:"b"}`),
		"unexpect8 object":  []byte(`{"a":"a","b":~"b"}`),
		"unexpect9 object":  []byte(`{"a":"a","b":"b"~}`),
		"unexpect10 object": []byte(`{"a":"a","b":"b"}~`),
		"invalid object":    []byte(`{1:1}`),
		"invalid2 object":   []byte(`{-1:1}`),
		"invalid3 object":   []byte(`{1.1:1}`),
		"invalid4 object":   []byte(`{true:1}`),
		"invalid5 object":   []byte(`{[]:1}`),
		"invalid6 object":   []byte(`{{}:1}`),
		"nested object": []byte(`{
			"arrays":	{
				"of int":		[1,2,3],
				"of string":	["a","b","c"],
				"of bool":		[true,false,true],
				"of anything":	["a",1,true]
			},
			"numbers":	{
				"one": 				1,
				"negative devil":	-666,
				"floaty":			6.3e-9
			},
			"string": "the cat sat on the mat",
			"deep": {
				"down": {
					"down": {
						"deeper": {
							"and": {
								"down": ":-O"
							}
						}
					}
				}
			}
		}`),
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
		destJ, dest interface{}
	}{
		"null_*interface{}": {[]byte(`null`), new(interface{}), new(interface{})},
		"null_interface{}":  {[]byte(`null`), nil, nil},
		"null_*string":      {[]byte(`null`), new(string), new(string)},
		"null_string":       {[]byte(`null`), "", ""},
		"null_*uint":        {[]byte(`null`), new(uint), new(uint)},
		"null_uint":         {[]byte(`null`), uint(0), uint(0)},
		"null_*int":         {[]byte(`null`), new(int), new(int)},
		"null_int":          {[]byte(`null`), 0, 0},
		"null_*float":       {[]byte(`null`), new(float64), new(float64)},
		"null_float":        {[]byte(`null`), float64(0), float64(0)},
		"null_*bool":        {[]byte(`null`), new(bool), new(bool)},
		"null_bool":         {[]byte(`null`), false, false},

		"string_*interface{}": {[]byte(`"string"`), new(interface{}), new(interface{})},
		"string_interface{}":  {[]byte(`"string"`), nil, nil},
		"string_*string":      {[]byte(`"string"`), new(string), new(string)},
		"string_string":       {[]byte(`"string"`), "", ""},
		"string_*int":         {[]byte(`"string"`), new(int), new(int)},
		"string_int":          {[]byte(`"string"`), 0, 0},

		"bool_*interface{}": {[]byte(`true`), new(interface{}), new(interface{})},
		"bool_interface{}":  {[]byte(`true`), nil, nil},
		"bool_*bool":        {[]byte(`true`), new(bool), new(bool)},
		"bool_bool":         {[]byte(`true`), false, false},
		"bool_*int":         {[]byte(`true`), new(int), new(int)},
		"bool_int":          {[]byte(`true`), 0, 0},

		"uint_*interface{}": {[]byte(`1`), new(interface{}), new(interface{})},
		"uint_interface{}":  {[]byte(`1`), nil, nil},
		"uint_*uint64":      {[]byte(`1`), new(uint64), new(uint64)},
		"uint_uint64":       {[]byte(`1`), uint64(0), uint64(0)},
		"uint_*uint32":      {[]byte(`1`), new(uint32), new(uint32)},
		"uint_uint32":       {[]byte(`1`), uint32(0), uint32(0)},
		"uint_*uint16":      {[]byte(`1`), new(uint16), new(uint16)},
		"uint_uint16":       {[]byte(`1`), uint16(0), uint16(0)},
		"uint_*uint8":       {[]byte(`1`), new(uint8), new(uint8)},
		"uint_uint8":        {[]byte(`1`), uint8(0), uint8(0)},
		"uint_*uint":        {[]byte(`1`), new(uint), new(uint)},
		"uint_uint":         {[]byte(`1`), uint(0), uint(0)},
		"uint_*int64":       {[]byte(`1`), new(int64), new(int64)},
		"uint_int64":        {[]byte(`1`), int64(0), int64(0)},
		"uint_*int32":       {[]byte(`1`), new(int32), new(int32)},
		"uint_int32":        {[]byte(`1`), int32(0), int32(0)},
		"uint_*int16":       {[]byte(`1`), new(int16), new(int16)},
		"uint_int16":        {[]byte(`1`), int16(0), int16(0)},
		"uint_*int8":        {[]byte(`1`), new(int8), new(int8)},
		"uint_int8":         {[]byte(`1`), int8(0), int8(0)},
		"uint_*int":         {[]byte(`1`), new(int), new(int)},
		"uint_int":          {[]byte(`1`), int(0), int(0)},
		"uint_*float64":     {[]byte(`1`), new(float64), new(float64)},
		"uint_float64":      {[]byte(`1`), float64(0), float64(0)},
		"uint_*float32":     {[]byte(`1`), new(float32), new(float32)},
		"uint_float32":      {[]byte(`1`), float32(0), float32(0)},
		"uint_*string":      {[]byte(`1`), new(string), new(string)},
		"uint_string":       {[]byte(`1`), "", ""},

		"int_*interface{}": {[]byte(`-1`), new(interface{}), new(interface{})},
		"int_interface{}":  {[]byte(`-1`), nil, nil},
		"int_*uint64":      {[]byte(`-1`), new(uint64), new(uint64)},
		"int_uint64":       {[]byte(`-1`), uint64(0), uint64(0)},
		"int_*uint32":      {[]byte(`-1`), new(uint32), new(uint32)},
		"int_uint32":       {[]byte(`-1`), uint32(0), uint32(0)},
		"int_*uint16":      {[]byte(`-1`), new(uint16), new(uint16)},
		"int_uint16":       {[]byte(`-1`), uint16(0), uint16(0)},
		"int_*uint8":       {[]byte(`-1`), new(uint8), new(uint8)},
		"int_uint8":        {[]byte(`-1`), uint8(0), uint8(0)},
		"int_*uint":        {[]byte(`-1`), new(uint), new(uint)},
		"int_uint":         {[]byte(`-1`), uint(0), uint(0)},
		"int_*int64":       {[]byte(`-1`), new(int64), new(int64)},
		"int_int64":        {[]byte(`-1`), int64(0), int64(0)},
		"int_*int32":       {[]byte(`-1`), new(int32), new(int32)},
		"int_int32":        {[]byte(`-1`), int32(0), int32(0)},
		"int_*int16":       {[]byte(`-1`), new(int16), new(int16)},
		"int_int16":        {[]byte(`-1`), int16(0), int16(0)},
		"int_*int8":        {[]byte(`-1`), new(int8), new(int8)},
		"int_int8":         {[]byte(`-1`), int8(0), int8(0)},
		"int_*int":         {[]byte(`-1`), new(int), new(int)},
		"int_int":          {[]byte(`-1`), int(0), int(0)},
		"int_*float64":     {[]byte(`-1`), new(float64), new(float64)},
		"int_float64":      {[]byte(`-1`), float64(0), float64(0)},
		"int_*float32":     {[]byte(`-1`), new(float32), new(float32)},
		"int_float32":      {[]byte(`-1`), float32(0), float32(0)},
		"int_*string":      {[]byte(`-1`), new(string), new(string)},
		"int_string":       {[]byte(`-1`), "", ""},

		"float_*interface{}": {[]byte(`1.2`), new(interface{}), new(interface{})},
		"float_interface{}":  {[]byte(`1.2`), nil, nil},
		"float_*uint64":      {[]byte(`1.2`), new(uint64), new(uint64)},
		"float_uint64":       {[]byte(`1.2`), uint64(0), uint64(0)},
		"float_*uint32":      {[]byte(`1.2`), new(uint32), new(uint32)},
		"float_uint32":       {[]byte(`1.2`), uint32(0), uint32(0)},
		"float_*uint16":      {[]byte(`1.2`), new(uint16), new(uint16)},
		"float_uint16":       {[]byte(`1.2`), uint16(0), uint16(0)},
		"float_*uint8":       {[]byte(`1.2`), new(uint8), new(uint8)},
		"float_*uint8_long":  {[]byte(`12.3`), new(uint8), new(uint8)},
		"float_*uint8_vlong": {[]byte(`1234567890.2`), new(uint8), new(uint8)},
		"float_uint8":        {[]byte(`1.2`), uint8(0), uint8(0)},
		"float_*uint":        {[]byte(`1.2`), new(uint), new(uint)},
		"float_uint":         {[]byte(`1.2`), uint(0), uint(0)},
		"float_*int64":       {[]byte(`1.2`), new(int64), new(int64)},
		"float_int64":        {[]byte(`1.2`), int64(0), int64(0)},
		"float_*int32":       {[]byte(`1.2`), new(int32), new(int32)},
		"float_int32":        {[]byte(`1.2`), int32(0), int32(0)},
		"float_*int16":       {[]byte(`1.2`), new(int16), new(int16)},
		"float_int16":        {[]byte(`1.2`), int16(0), int16(0)},
		"float_*int8":        {[]byte(`1.2`), new(int8), new(int8)},
		"float_int8":         {[]byte(`1.2`), int8(0), int8(0)},
		"float_*int":         {[]byte(`1.2`), new(int), new(int)},
		"float_int":          {[]byte(`1.2`), int(0), int(0)},
		"float_*float64":     {[]byte(`1.2`), new(float64), new(float64)},
		"float_float64":      {[]byte(`1.2`), float64(0), float64(0)},
		"float_*float32":     {[]byte(`1.2`), new(float32), new(float32)},
		"float_float32":      {[]byte(`1.2`), float32(0), float32(0)},
		"float_*string":      {[]byte(`1.2`), new(string), new(string)},
		"float_string":       {[]byte(`1.2`), "", ""},

		"negfloat_*interface{}": {[]byte(`-1.2`), new(interface{}), new(interface{})},
		"negfloat_interface{}":  {[]byte(`-1.2`), nil, nil},
		"negfloat_*uint64":      {[]byte(`-1.2`), new(uint64), new(uint64)},
		"negfloat_uint64":       {[]byte(`-1.2`), uint64(0), uint64(0)},
		"negfloat_*uint32":      {[]byte(`-1.2`), new(uint32), new(uint32)},
		"negfloat_uint32":       {[]byte(`-1.2`), uint32(0), uint32(0)},
		"negfloat_*uint16":      {[]byte(`-1.2`), new(uint16), new(uint16)},
		"negfloat_uint16":       {[]byte(`-1.2`), uint16(0), uint16(0)},
		"negfloat_*uint8":       {[]byte(`-1.2`), new(uint8), new(uint8)},
		"negfloat_*uint8_long":  {[]byte(`-12.3`), new(uint8), new(uint8)},
		"negfloat_*uint8_vlong": {[]byte(`-1234567890.2`), new(uint8), new(uint8)},
		"negfloat_uint8":        {[]byte(`-1.2`), uint8(0), uint8(0)},
		"negfloat_*uint":        {[]byte(`-1.2`), new(uint), new(uint)},
		"negfloat_uint":         {[]byte(`-1.2`), uint(0), uint(0)},
		"negfloat_*int64":       {[]byte(`-1.2`), new(int64), new(int64)},
		"negfloat_int64":        {[]byte(`-1.2`), int64(0), int64(0)},
		"negfloat_*int32":       {[]byte(`-1.2`), new(int32), new(int32)},
		"negfloat_int32":        {[]byte(`-1.2`), int32(0), int32(0)},
		"negfloat_*int16":       {[]byte(`-1.2`), new(int16), new(int16)},
		"negfloat_int16":        {[]byte(`-1.2`), int16(0), int16(0)},
		"negfloat_*int8":        {[]byte(`-1.2`), new(int8), new(int8)},
		"negfloat_int8":         {[]byte(`-1.2`), int8(0), int8(0)},
		"negfloat_*int":         {[]byte(`-1.2`), new(int), new(int)},
		"negfloat_int":          {[]byte(`-1.2`), int(0), int(0)},
		"negfloat_*float64":     {[]byte(`-1.2`), new(float64), new(float64)},
		"negfloat_float64":      {[]byte(`-1.2`), float64(0), float64(0)},
		"negfloat_*float32":     {[]byte(`-1.2`), new(float32), new(float32)},
		"negfloat_float32":      {[]byte(`-1.2`), float32(0), float32(0)},
		"negfloat_*string":      {[]byte(`-1.2`), new(string), new(string)},
		"negfloat_string":       {[]byte(`-1.2`), "", ""},

		"[3]int_*interface{}": {[]byte(`[1,2,3]`), new(interface{}), new(interface{})},
		"[3]int_interface{}":  {[]byte(`[1,2,3]`), nil, nil},
		"[3]int_*[]int":       {[]byte(`[1,2,3]`), new([]int), new([]int)},
		"[3]int_[]int":        {[]byte(`[1,2,3]`), []int{}, []int{}},
		"[3]int_*[](2)int": {
			[]byte(`[1,2,3]`),
			func() *[]int { i := make([]int, 2); return &i }(),
			func() *[]int { i := make([]int, 2); return &i }(),
		},
		"[3]int_[](2)int": {
			[]byte(`[1,2,3]`),
			make([]int, 2),
			make([]int, 2)},
		"[3]int_*[](3)int": {
			[]byte(`[1,2,3]`),
			func() *[]int { i := make([]int, 3); return &i }(),
			func() *[]int { i := make([]int, 3); return &i }(),
		},
		"[3]int_[](3)int": {
			[]byte(`[1,2,3]`),
			make([]int, 3),
			make([]int, 3)},
		"[3]int_*[](4)int": {
			[]byte(`[1,2,3]`),
			func() *[]int { i := make([]int, 4); return &i }(),
			func() *[]int { i := make([]int, 4); return &i }(),
		},
		"[3]int_[](4)int": {
			[]byte(`[1,2,3]`),
			make([]int, 4),
			make([]int, 4)},
		"[3]int_*[](0,2)int": {
			[]byte(`[1,2,3]`),
			func() *[]int { i := make([]int, 0, 2); return &i }(),
			func() *[]int { i := make([]int, 0, 2); return &i }(),
		},
		"[3]int_[](0,2)int": {
			[]byte(`[1,2,3]`),
			make([]int, 0, 2),
			make([]int, 0, 2)},
		"[3]int_*[](0,3)int": {
			[]byte(`[1,2,3]`),
			func() *[]int { i := make([]int, 0, 3); return &i }(),
			func() *[]int { i := make([]int, 0, 3); return &i }(),
		},
		"[3]int_[](0,3)int": {
			[]byte(`[1,2,3]`),
			make([]int, 0, 3),
			make([]int, 0, 3)},
		"[3]int_*[](0,4)int": {
			[]byte(`[1,2,3]`),
			func() *[]int { i := make([]int, 0, 4); return &i }(),
			func() *[]int { i := make([]int, 0, 4); return &i }(),
		},
		"[3]int_[](0,4)int": {[]byte(`[1,2,3]`), make([]int, 0, 4), make([]int, 0, 4)},
		"[3]int_*[2]int":    {[]byte(`[1,2,3]`), new([2]int), new([2]int)},
		"[3]int_[2]int":     {[]byte(`[1,2,3]`), [2]int{}, [2]int{}},
		"[3]int_*[3]int":    {[]byte(`[1,2,3]`), new([3]int), new([3]int)},
		"[3]int_[3]int":     {[]byte(`[1,2,3]`), [3]int{}, [3]int{}},
		"[3]int_*[4]int":    {[]byte(`[1,2,3]`), new([4]int), new([4]int)},
		"[3]int_[4]int":     {[]byte(`[1,2,3]`), [4]int{}, [4]int{}},
		"[3]int_*[]string":  {[]byte(`[1,2,3]`), new([]string), new([]string)},
		"[3]int_[]string":   {[]byte(`[1,2,3]`), []string{}, []string{}},
		"[3]int_*int":       {[]byte(`[1,2,3]`), new(int), new(int)},
		"[3]int_int":        {[]byte(`[1,2,3]`), 0, 0},
		// This exercises inputs too large for destination arrays, the addional 2 arrays help with a difference in how
		// encoding/json assings capacity in the slices
		"[][4]int_*[]2int":      {[]byte(`[[1,2,3,4],[5,6,7,8],[],[]]`), new([][2]int), new([][2]int)},
		"[3]float_*[]int":       {[]byte(`[1.2,1.2,1.3]`), new([]int), new([]int)},
		"[1][1]int_*[][]string": {[]byte(`[[1]]`), new([][]string), new([][]string)},

		// TODO deep pointers []*imt, *******int and so on.
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Log(string(tt.input))
			errJ := json.NewDecoder(bytes.NewBuffer(tt.input)).Decode(tt.destJ)
			err := NewDecoder(bytes.NewBuffer(tt.input)).Decode(tt.dest)
			if errJ == nil {
				// There are differences when errJ != nil, such as in test: '[3]int_*[]string'
				t.Log("Expected: ", tt.destJ)
				t.Log("Actual  : ", tt.dest)
				assert.Equal(t, tt.destJ, tt.dest)
				if reflect.ValueOf(tt.destJ).Kind() == reflect.Ptr && reflect.ValueOf(tt.destJ).Elem().Kind() == reflect.Slice {
					assert.Equal(t, reflect.ValueOf(tt.destJ).Elem().Cap(), reflect.ValueOf(tt.dest).Elem().Cap(), "bad capacity")
				}
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
		"null":        `n`,
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
		"arr":         `[`,
		"arr2":        `[" "`,
		"obj":         `{`,
		"objkey":      `{"a"`,
		"objsep":      `{"a":`,
		"objval":      `{"a":"a"`,
		"objspace":    `{ `,
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
	t.Log("expected error: ", expected)
	t.Log("actual error  : ", err)
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
