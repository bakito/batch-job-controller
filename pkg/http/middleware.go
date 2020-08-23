package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *PostServer) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Cache != nil {
			vars := mux.Vars(r)
			executionID := vars["executionID"]
			if !s.Cache.Has(executionID) {
				http.Error(w, "execution ID not allowed", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
