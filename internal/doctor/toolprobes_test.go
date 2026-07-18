package doctor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestProbePayloadEmpty(t *testing.T) {
	if !probePayloadEmpty(nil) {
		t.Fatal("nil should be empty")
	}
	if !probePayloadEmpty([]any{}) {
		t.Fatal("empty slice should be empty")
	}
	if probePayloadEmpty([]any{"x"}) {
		t.Fatal("non-empty slice should not be empty")
	}
}

func TestProbeSearchCodeOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/searchCode" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"code": "00700.HK", "name": "腾讯控股"}})
	}))
	defer srv.Close()

	cfg := &config.AppConfig{
		SignalAPIURLField:  srv.URL,
		SignalAPIKeyField:  "sk-test",
		Sandbox:            config.SandboxConfig{AllowedHosts: []string{"127.0.0.1"}},
	}
	row := probeSearchCode(t.Context(), cfg)
	if !row.OK || row.Warn {
		t.Fatalf("expected OK without warn, got %+v", row)
	}
}

func TestProbeSearchCodeWarnEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	cfg := &config.AppConfig{
		SignalAPIURLField: srv.URL,
		Sandbox:           config.SandboxConfig{AllowedHosts: []string{"127.0.0.1"}},
	}
	row := probeSearchCode(t.Context(), cfg)
	if !row.OK || !row.Warn {
		t.Fatalf("expected WARN, got %+v", row)
	}
}

func TestAnyWarned(t *testing.T) {
	if anyWarned([]CheckResult{{OK: true, Warn: true}}) != true {
		t.Fatal("expected warned")
	}
	if anyWarned([]CheckResult{{OK: true}}) {
		t.Fatal("expected not warned")
	}
}
