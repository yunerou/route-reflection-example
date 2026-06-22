package muxmiddleware

import (
	"net/http"
	"strings"

	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/encdec"
)

const (
	contentTypeMsgpack = "application/msgpack"
	contentTypeJSON    = "application/json; charset=utf-8"
)

// wantsMsgpack returns true when the request Accept header includes msgpack.
func wantsMsgpack(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, contentTypeMsgpack)
}

// EncoderDecoder picks an encoder/decoder pair based on the request Accept
// header (msgpack when wantsMsgpack reports true, otherwise JSON), stores
// them in actx, and writes the matching Content-Type response header so
// downstream handlers do not need to inspect the request again.
func (mw *middlewareProvider) EncoderDecoder() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				enc         encdec.Encoder
				dec         encdec.Decoder
				contentType string
			)
			if wantsMsgpack(r) {
				enc = encdec.MsgpackEncoder()
				dec = encdec.MsgpackDecoder()
				contentType = contentTypeMsgpack
			} else {
				enc = encdec.JSONEncoder()
				dec = encdec.JSONDecoder()
				contentType = contentTypeJSON
			}

			aCtx := actx.From(r.Context())
			aCtx.SetEncoderDecoder(enc, dec)

			w.Header().Set("Content-Type", contentType)

			next.ServeHTTP(w, r.WithContext(aCtx))
		})
	}
}
