// Package usernotice syncs Agent gateway state into QT_DB.user.notice.
package usernotice

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/gateway/feishustore"
)

const (
	ChannelFeishuIM = "feishu_im"
	ChannelWebhook  = "webhook"
	ChannelWechat   = "wechat"

	defaultBotMongoDB = "QT_DB"
)

// FeishuSection mirrors QT_DB.user.notice.feishu (no secrets).
type FeishuSection struct {
	Enabled    bool   `bson:"enabled" json:"enabled"`
	Connected  bool   `bson:"connected" json:"connected"`
	ReceiveID  string `bson:"receive_id,omitempty" json:"receive_id,omitempty"`
	BotName    string `bson:"bot_name,omitempty" json:"bot_name,omitempty"`
	AppID      string `bson:"app_id,omitempty" json:"app_id,omitempty"`
	UpdatedAt  string `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// SyncFeishuGateway writes feishu_im channel metadata after gateway setup.
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
	section := FeishuSection{
		Enabled:   connected,
		Connected: connected,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if creds != nil {
		section.BotName = strings.TrimSpace(creds.BotName)
		section.AppID = strings.TrimSpace(creds.AppID)
		section.ReceiveID = resolveReceiveID(creds)
	}

	set := bson.M{
		"notice.feishu": section,
	}
	if connected {
		set["notice.channel"] = ChannelFeishuIM
		set["notice.notice_type"] = 1
		set["notice.notice_url"] = ""
	}

	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer func() { _ = cli.Disconnect(context.Background()) }()

	update := bson.M{"$set": set}
	if connected {
		update["$unset"] = bson.M{
			"notice_url":  "",
			"notice_type": "",
		}
	}
	_, err = cli.Database(dbName).Collection("user").UpdateOne(
		ctx,
		bson.M{"_id": oid},
		update,
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
