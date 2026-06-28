package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Options configures the HTTP server.
type Options struct {
	Name            string
	Port            int
	ReadTimeoutSec  int
	WriteTimeoutSec int
}

// NewMux returns a mux with GET /health and GET /ready.
func NewMux(serviceName string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": serviceName,
		})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ready",
			"service": serviceName,
		})
	})
	return mux
}

// New builds an http.Server with long timeouts for LLM upstream (≥310s).
func New(opts Options, handler http.Handler) *http.Server {
	readTO := opts.ReadTimeoutSec
	if readTO == 0 {
		readTO = 320
	}
	writeTO := opts.WriteTimeoutSec
	if writeTO == 0 {
		writeTO = 320
	}
	if handler == nil {
		handler = NewMux(opts.Name)
	}
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", opts.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(readTO) * time.Second,
		WriteTimeout: time.Duration(writeTO) * time.Second,
	}
}

// Run listens until ctx is cancelled.
func Run(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		slog.Info("agent-runtime listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
