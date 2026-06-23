package muxrouter

import (
	"io"

	"github.com/bytedance/sonic"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	MsgPackHeaders = []string{"application/x-msgpack", "application/msgpack"}
	MsgPackFormat  = Format{
		Marshal: func(w io.Writer, v any) error {
			e := msgpack.NewEncoder(w)
			e.SetCustomStructTag("json")

			return e.Encode(v)
		},
		Unmarshal: func(r io.Reader, v any) error {
			d := msgpack.NewDecoder(r)
			d.SetCustomStructTag("json")
			return d.Decode(v)
		},
	}
	JsonHeaders     = []string{"application/json", "text/json"}
	JsonSonicFormat = Format{
		Marshal: func(w io.Writer, v any) error {
			dec := sonic.ConfigDefault.NewEncoder(w)
			return dec.Encode(v)
		},
		Unmarshal: func(r io.Reader, v any) error {
			dec := sonic.ConfigDefault.NewDecoder(r)
			return dec.Decode(v)
		},
	}
)

// Register Format type
type Format struct {
	Marshal   func(w io.Writer, v any) error
	Unmarshal func(r io.Reader, v any) error
}

type RegisterFormat struct {
	Headers []string // List of content-type headers to match for this format
	Formats Format
}

// Register Error type
// Register Error type
type StatusError interface {
	GetStatus() int
	Error() string
}
