package feishupush

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

// UserOpts configures a per-user Feishu digest send.
type UserOpts struct {
	Workspace string
	UserID    string
	Text      string
	DB        *infra.DB
	PG        *infra.PostgresDB
}

// SendUser delivers a digest using the user's own Feishu bot credentials.
func SendUser(ctx context.Context, opts UserOpts) error {
	text := strings.TrimSpace(opts.Text)
	userID := strings.TrimSpace(opts.UserID)
	if text == "" {
		return fmt.Errorf("feishupush: empty message")
	}
	if userID == "" {
		return fmt.Errorf("feishupush: empty user_id")
	}
	dir := strings.TrimSpace(opts.Workspace)
	if dir == "" {
		dir = "."
	}
	creds, err := feishustore.Load(dir, userID)
	if err != nil {
		return err
	}
	if creds == nil || !creds.Configured() {
		return fmt.Errorf("feishupush: feishu creds missing for user %s", userID)
	}
	if !creds.Enabled {
		return fmt.Errorf("feishupush: feishu disabled for user %s", userID)
	}
	receiveID, err := ResolveUserReceiveID(ctx, creds, openSessionDB(opts), opts.PG, userID)
	if err != nil {
		return err
	}
	if err := sendMarkdown(ctx, *creds, receiveID, text); err != nil {
		return err
	}
	slog.Info("feishupush: user digest sent", "user_id", userID, "chars", len(text))
	return nil
}

// SendUserWithRetry retries once after a short backoff.
func SendUserWithRetry(ctx context.Context, opts UserOpts) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := SendUser(ctx, opts); err != nil {
			lastErr = err
			slog.Warn("feishupush: send failed", "user_id", opts.UserID, "attempt", attempt+1, "err", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func sendMarkdown(ctx context.Context, creds feishustore.Creds, receiveID, text string) error {
	domain := creds.Domain
	if domain == "" {
		domain = "feishu"
	}
	baseURL := lark.FeishuBaseUrl
	if strings.EqualFold(domain, "lark") {
		baseURL = lark.LarkBaseUrl
	}
	api := lark.NewClient(creds.AppID, creds.AppSecret, lark.WithOpenBaseUrl(baseURL))
	postContent := BuildMarkdownPostPayload(text)
	if _, err := createMessage(ctx, api, receiveID, "post", postContent); err != nil {
		if !isPostContentInvalid(err) {
			return err
		}
		textContent, _ := json.Marshal(map[string]string{"text": PlainTextFallback(text)})
		_, err = createMessage(ctx, api, receiveID, "text", string(textContent))
		return err
	}
	return nil
}

func createMessage(ctx context.Context, api *lark.Client, receiveID, msgType, content string) (string, error) {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType(receiveID)).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(strings.TrimSpace(receiveID)).
			MsgType(msgType).
			Content(content).
			Build()).
		Build()
	resp, err := api.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu send: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", nil
	}
	return *resp.Data.MessageId, nil
}

func isPostContentInvalid(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "content format of the post type is incorrect") ||
		strings.Contains(s, "post type is incorrect")
}

// ResolveUserReceiveID finds the Feishu DM target for a GeeGoo user.
func ResolveUserReceiveID(ctx context.Context, creds *feishustore.Creds, db *infra.DB, pg *infra.PostgresDB, ownerUserID string) (string, error) {
	if creds == nil {
		return "", fmt.Errorf("feishupush: nil creds")
	}
	for _, id := range creds.AllowedUsers {
		if isUserOpenID(id) {
			return strings.TrimSpace(id), nil
		}
	}
	if isUserOpenID(creds.HomeChannel) {
		return strings.TrimSpace(creds.HomeChannel), nil
	}
	if id, ok := lookupFeishuOpenID(ctx, db, pg, ownerUserID); ok {
		return id, nil
	}
	return "", fmt.Errorf("feishupush: no receive_id for user %s (set allowed_users or chat with bot once)", ownerUserID)
}

func isUserOpenID(id string) bool {
	return strings.HasPrefix(strings.TrimSpace(id), "ou_")
}

func receiveIDType(id string) string {
	id = strings.TrimSpace(id)
	switch {
	case strings.HasPrefix(id, "ou_"):
		return larkim.CreateMessageV1ReceiveIDTypeOpenId
	case strings.HasPrefix(id, "on_"):
		return larkim.CreateMessageV1ReceiveIDTypeUnionId
	case strings.Contains(id, "@"):
		return larkim.CreateMessageV1ReceiveIDTypeEmail
	default:
		return larkim.CreateMessageV1ReceiveIDTypeChatId
	}
}

func lookupFeishuOpenID(ctx context.Context, db *infra.DB, pg *infra.PostgresDB, ownerUserID string) (string, bool) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return "", false
	}
	if pg != nil && pg.SQL() != nil {
		if id, ok := scanSessionMetadata(ctx, pg.SQL(), `
			SELECT metadata_json::text FROM chat_sessions
			WHERE user_id = $1
			ORDER BY updated_at DESC
			LIMIT 40`, ownerUserID); ok {
			return id, true
		}
	}
	if db != nil && db.SQL() != nil {
		if id, ok := scanSessionMetadata(ctx, db.SQL(), `
			SELECT metadata_json FROM chat_sessions
			ORDER BY updated_at DESC
			LIMIT 500`, ownerUserID); ok {
			return id, true
		}
	}
	return "", false
}

func scanSessionMetadata(ctx context.Context, conn *sql.DB, query string, args ...any) (string, bool) {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		if id, ok := feishuOpenIDFromMetadata(raw, args...); ok {
			return id, true
		}
	}
	return "", false
}

func feishuOpenIDFromMetadata(raw string, filterArgs ...any) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return "", false
	}
	var meta map[string]any
	if json.Unmarshal([]byte(raw), &meta) != nil {
		return "", false
	}
	if len(filterArgs) > 0 {
		if want, ok := filterArgs[0].(string); ok && want != "" {
			owner := metadataString(meta, "gateway_owner_user_id")
			if owner == "" {
				owner = metadataString(meta, "user_id")
			}
			if owner != "" && owner != want {
				return "", false
			}
		}
	}
	if id := metadataString(meta, "gateway_feishu_user"); isUserOpenID(id) {
		return id, true
	}
	return "", false
}

func metadataString(meta map[string]any, key string) string {
	v, ok := meta[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func openSessionDB(opts UserOpts) *infra.DB {
	if opts.DB != nil {
		return opts.DB
	}
	dir := strings.TrimSpace(opts.Workspace)
	if dir == "" {
		dir = "."
	}
	path := filepath.Join(dir, "geegoo.db")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	db, err := infra.OpenSQLite(path)
	if err != nil {
		slog.Warn("feishupush: open sqlite for receive_id lookup failed", "path", path, "err", err)
		return nil
	}
	return db
}
