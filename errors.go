package json

import (
	"fmt"
	"reflect"
)

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (i *InvalidUnmarshalError) Error() string {
	if i.Type == nil {
		return "json: Unmarshal(nil)"
	}
	return "json: Unmarshal(non-pointer " + i.Type.String() + ")"
}

type SyntaxError struct {
	msg    string
	Offset int64
}

func (d *Decoder) syntaxErrorf(format string, a ...interface{}) *SyntaxError {
	return &SyntaxError{
		msg:    fmt.Sprintf(format, a...),
		Offset: d.offset,
	}
}

func (s *SyntaxError) Error() string {
	return s.msg
}

type UnmarshalTypeError struct {
	Value  string
	Type   reflect.Type
	Offset int64
	Struct string
	Field  string
}

func (d *Decoder) unmarshalTypeError(value string, t reflect.Type) *UnmarshalTypeError {
	return &UnmarshalTypeError{
		Value:  value,
		Type:   t,
		Offset: d.offset,
	}
}

func (u *UnmarshalTypeError) Error() string {
	return "json: cannot unmarshal " + u.Value + " into Go value of type " + u.Type.String()
}
