package http

import (
	"net/http"
)

const (
	errorMiddlewareNotAcceptable = "node / execution ID not allowed"
)

func (s *PostServer) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Controller != nil {
			if !s.Controller.Has(s.nodeAndID(r)) {
				http.Error(w, errorMiddlewareNotAcceptable, http.StatusNotAcceptable)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
