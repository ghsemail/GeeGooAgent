package runtimeapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
)

func TestContextProfilesGetAndPut(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GEEGOO_HOME", home)
	h := &Handler{App: &app.App{}}
	mux := http.NewServeMux()
	h.registerContextProfileRoutes(mux)

	putReq := httptest.NewRequest(http.MethodPut, "/v1/context/profiles/user_default/_", bytes.NewReader([]byte(`{"content":"- prefer concise replies\n"}`)))
	putReq.Header.Set("X-User-Id", "u1")
	putReq.SetPathValue("kind", "user_default")
	putReq.SetPathValue("key", "_")
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/context/profiles/inspect?merged=1", nil)
	getReq.Header.Set("X-User-Id", "u1")
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "prefer concise replies") {
		t.Fatalf("body=%s", getRec.Body.String())
	}

	liteReq := httptest.NewRequest(http.MethodGet, "/v1/context/profiles/inspect", nil)
	liteReq.Header.Set("X-User-Id", "u1")
	liteRec := httptest.NewRecorder()
	mux.ServeHTTP(liteRec, liteReq)
	if liteRec.Code != http.StatusOK {
		t.Fatalf("lite status=%d body=%s", liteRec.Code, liteRec.Body.String())
	}
	if !strings.Contains(liteRec.Body.String(), `"scope":"user"`) {
		t.Fatalf("lite body=%s", liteRec.Body.String())
	}
}
