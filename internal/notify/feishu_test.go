package notify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/notify"
)

func TestFeishuSenderWebhook(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	sender := notify.NewFeishuSender(srv.URL)
	if err := sender.Send(context.Background(), "hello summary"); err != nil {
		t.Fatal(err)
	}
	if got["msg_type"] != "text" {
		t.Fatalf("msg_type=%v", got["msg_type"])
	}
}
