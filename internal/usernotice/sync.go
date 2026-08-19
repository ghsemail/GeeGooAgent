// Package usernotice syncs Agent gateway state into QT_DB.user.notice.
package usernotice

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
)

const defaultBotMongoDB = "QT_DB"

// FeishuSection mirrors QT_DB.user.notice.feishu (no secrets).
type FeishuSection struct {
	Connected bool   `bson:"connected" json:"connected"`
	ReceiveID string `bson:"receive_id,omitempty" json:"receive_id,omitempty"`
	BotName   string `bson:"bot_name,omitempty" json:"bot_name,omitempty"`
}

var legacyNoticeUnset = bson.M{
	"notice_url":                 "",
	"notice_type":                "",
	"notice.channel":             "",
	"notice.notice_url":          "",
	"notice.notice_type":         "",
	"notice.feishu.enabled":    "",
	"notice.feishu.app_id":       "",
	"notice.feishu.updated_at":   "",
}

// SyncFeishuGateway writes notice.feishu after gateway setup.
func SyncFeishuGateway(ctx context.Context, cfg *config.AppConfig, userID string, creds *feishustore.Creds) error {
	if cfg == nil {
		return errors.New("usernotice: nil config")
	}
	uri := strings.TrimSpace(cfg.BotMongoURI)
	if uri == "" {
		return errors.New("usernotice: bot mongo not configured")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("usernotice: empty user id")
	}
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	dbName := strings.TrimSpace(cfg.BotMongoDB)
	if dbName == "" {
		dbName = defaultBotMongoDB
	}

	connected := creds != nil && creds.Configured() && creds.Enabled
	section := FeishuSection{Connected: connected}
	if creds != nil {
		section.BotName = strings.TrimSpace(creds.BotName)
		section.ReceiveID = resolveReceiveID(creds)
	}

	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer func() { _ = cli.Disconnect(context.Background()) }()

	_, err = cli.Database(dbName).Collection("user").UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{
			"$set":   bson.M{"notice.feishu": section},
			"$unset": legacyNoticeUnset,
		},
	)
	return err
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
