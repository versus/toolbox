package xhttp

import (
	"fmt"
	"net/http"
)

// PasswordLookup provides a way to map a user in a realm to a password
type PasswordLookup func(user string, realm string) string

// BasicAuth provides basic HTTP authentication.
type BasicAuth struct {
	realm  string
	lookup PasswordLookup
}

// NewBasicAuth creates a new BasicAuth.
func NewBasicAuth(realm string, lookup PasswordLookup) *BasicAuth {
	return &BasicAuth{realm: realm, lookup: lookup}
}

// Wrap an http.Handler.
func (auth *BasicAuth) Wrap(handler http.Handler) http.Handler {
	return &wrapper{auth: auth, handler: handler}
}

type wrapper struct {
	auth    *BasicAuth
	handler http.Handler
}

func (hw *wrapper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if user, pw, ok := req.BasicAuth(); ok {
		if pw == hw.auth.lookup(user, hw.auth.realm) {
			hw.handler.ServeHTTP(w, req)
			return
		}
	}
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, hw.auth.realm))
	WriteHTTPStatus(w, http.StatusUnauthorized)
}
