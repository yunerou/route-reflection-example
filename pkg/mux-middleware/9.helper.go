package muxmiddleware

import (
	"net/http"
	"strings"
)

// wantsMsgpack returns true when the request Accept header includes msgpack.
func wantsMsgpack(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/msgpack") ||
		strings.Contains(accept, "application/x-msgpack")
}
