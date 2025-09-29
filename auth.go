package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
)

func (h *kuttaHandler) wrapWithAuth(next http.Handler) http.Handler {
	if !h.AuthEnabled {
		return next
	}

	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte(h.AuthCreds))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != expected {
			w.Header().Set("WWW-Authenticate", `Basic realm="Kutta File Server"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("Unauthorized access to %s from %s", r.URL.Path, r.RemoteAddr)
			return
		}
		next.ServeHTTP(w, r)
	})
}
