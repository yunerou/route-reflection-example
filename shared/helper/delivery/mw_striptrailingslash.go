package delivery

import (
	"net/http"
	"strings"
)

// StripTrailingSlash removes trailing slashes from request paths before
// dispatching to the next handler. Useful with http.ServeMux so that
// "/api/foo/" is treated the same as "/api/foo".
func StripTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; len(p) > 1 && strings.HasSuffix(p, "/") {
			r.URL.Path = strings.TrimRight(p, "/")
		}
		next.ServeHTTP(w, r)
	})
}
