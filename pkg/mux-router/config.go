package muxrouter

import (
	"io"

	"github.com/bytedance/sonic"
	"github.com/vmihailenco/msgpack/v5"
)

// Format defines how a body is (de)serialized.
type Format struct {
	Marshal   func(w io.Writer, v any) error
	Unmarshal func(r io.Reader, v any) error
}

// RegisterFormat binds a Format to the content-type headers that select it.
type RegisterFormat struct {
	Headers []string
	Formats Format
}

// StatusError is an error carrying an HTTP status. Matches huma.StatusError.
type StatusError interface {
	GetStatus() int
	Error() string
}

// Config is shared by every adapter so application code is build-tag agnostic.
type Config struct {
	Formats      []RegisterFormat
	ConvertError func(error) StatusError
	CommonInfo   CommonInfo
	Doc          DocConfig // huma only; gomux ignores
}

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
			return sonic.ConfigDefault.NewEncoder(w).Encode(v)
		},
		Unmarshal: func(r io.Reader, v any) error {
			return sonic.ConfigDefault.NewDecoder(r).Decode(v)
		},
	}
)
