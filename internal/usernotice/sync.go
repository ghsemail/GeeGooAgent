// Package usernotice syncs Agent gateway state into QT_DB.user.notice.
package usernotice

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
)

// SyncFeishuGateway writes notice.feishu (including app_secret) after gateway setup.
func SyncFeishuGateway(ctx context.Context, cfg *config.AppConfig, userID string, creds *feishustore.Creds) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errEmptyUserID
	}
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	coll, disconnect, err := userColl(ctx, cfg)
	if err != nil {
		return err
	}
	defer disconnect()

	section := credsToSection(creds)
	_, err = coll.UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{
			"$set":   bson.M{"notice.feishu": section},
			"$unset": legacyNoticeUnset,
		},
	)
	return err
}

// LoadOrMigrateCreds reads from Mongo; one-time imports from the legacy file store if needed.
func LoadOrMigrateCreds(ctx context.Context, cfg *config.AppConfig, legacyFileDir, userID string) (*feishustore.Creds, error) {
	c, err := LoadCreds(ctx, cfg, userID)
	if err != nil {
		return nil, err
	}
	if c != nil && c.Configured() {
		return c, nil
	}
	fc, err := feishustore.Load(legacyFileDir, userID)
	if err != nil {
		return nil, err
	}
	if fc == nil || !fc.Configured() {
		return c, nil
	}
	if err := SyncFeishuGateway(ctx, cfg, userID, fc); err != nil {
		return fc, err
	}
	return fc, nil
}

func resolveReceiveID(creds *feishustore.Creds) string {
	if creds == nil {
		return ""
	}
	for _, id := range creds.AllowedUsers {
		id = strings.TrimSpace(id)
		if strings.HasPrefix(id, "ou_") {
			return id
		}
	}
	if id := strings.TrimSpace(creds.HomeChannel); strings.HasPrefix(id, "ou_") {
		return id
	}
	return ""
}

var errEmptyUserID = errString("usernotice: empty user id")

type errString string

func (e errString) Error() string { return string(e) }
