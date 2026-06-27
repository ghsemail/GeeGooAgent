package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(apiError{Code: 401, Message: "Unauthorized"})
}

// BearerAPIKey validates Authorization: Bearer <key>. Skips when expectedKey is empty.
func BearerAPIKey(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) || strings.TrimSpace(auth[len(prefix):]) != expectedKey {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SkipPaths exempts exact paths from Bearer auth.
func SkipPaths(paths map[string]struct{}, bearer func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		protected := bearer(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := paths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}
