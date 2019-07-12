package authentication

import (
	"net/http"
)

type Authentication struct {
	ServerPassword string
	ServerUsername string
}

func (a Authentication) getPassword() string {
	return a.ServerPassword
}

func (a Authentication) getUsername() string {
	return a.ServerUsername
}

func New(username string, password string) *Authentication {
	authentication := &Authentication{
		ServerPassword: password,
		ServerUsername: username,
	}
	return authentication
}

func (a Authentication) basicAuth(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()

		if a.ServerUsername != user || a.ServerPassword != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (a Authentication) WrapHandler(h http.HandlerFunc) http.HandlerFunc {
	return a.basicAuth(h)
}
