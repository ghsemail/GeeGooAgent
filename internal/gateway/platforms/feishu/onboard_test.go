package feishu_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/gateway/platforms/feishu"
)

func TestOnboardFlowWithMockServer(t *testing.T) {
	pollN := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/v1/app/registration", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.Form.Get("action")
		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "init":
			_ = json.NewEncoder(w).Encode(map[string]any{"supported_auth_methods": []string{"client_secret"}})
		case "begin":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code":               "dev-1",
				"verification_uri_complete": "https://example.test/scan",
				"user_code":                 "ABCD",
				"interval":                  1,
				"expire_in":                 30,
			})
		case "poll":
			pollN++
			if pollN < 2 {
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"client_id":     "cli_test",
				"client_secret": "sec_test",
				"user_info":     map[string]any{"open_id": "ou_user", "tenant_brand": "feishu"},
			})
		default:
			http.Error(w, "bad action", 400)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	old := feishu.OnboardHTTPClient
	feishu.OnboardHTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: rewriteHost{to: srv.URL, base: http.DefaultTransport},
	}
	t.Cleanup(func() { feishu.OnboardHTTPClient = old })

	ctx := context.Background()
	if err := feishu.InitRegistration(ctx, "feishu"); err != nil {
		t.Fatal(err)
	}
	begin, err := feishu.BeginRegistration(ctx, "feishu")
	if err != nil {
		t.Fatal(err)
	}
	if begin.DeviceCode != "dev-1" {
		t.Fatalf("%+v", begin)
	}
	if !strings.Contains(begin.QRURL, "from=geegoo") {
		t.Fatalf("qr url missing tag: %s", begin.QRURL)
	}
	creds, err := feishu.PollRegistration(ctx, begin.DeviceCode, 1, 10, "feishu")
	if err != nil {
		t.Fatal(err)
	}
	if creds.AppID != "cli_test" || creds.AppSecret != "sec_test" || creds.OpenID != "ou_user" {
		t.Fatalf("%+v", creds)
	}
}

type rewriteHost struct {
	to   string
	base http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := http.NewRequest(req.Method, r.to+req.URL.RequestURI(), req.Body)
	if err != nil {
		return nil, err
	}
	u.Header = req.Header.Clone()
	return r.base.RoundTrip(u)
}
