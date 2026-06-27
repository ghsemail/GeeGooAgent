package httpserver

import (
	"net/http"

	"github.com/ghsemail/GeeGooAgent/internal/auth"
)

// NewProtectedHandler wraps mux with Bearer auth; /health is public.
func NewProtectedHandler(serviceName, apiKey string, allowInsecure bool, register func(*http.ServeMux)) http.Handler {
	mux := NewMux(serviceName)
	if register != nil {
		register(mux)
	}
	key := apiKey
	if allowInsecure {
		key = ""
	}
	skip := map[string]struct{}{"/health": {}}
	return auth.SkipPaths(skip, auth.BearerAPIKey(key))(mux)
}
