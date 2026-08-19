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

// FeishuSection is QT_DB.user.notice.feishu — single source for Feishu bot creds.
type FeishuSection struct {
	Connected    bool     `bson:"connected" json:"connected"`
	AppID        string   `bson:"app_id,omitempty" json:"app_id,omitempty"`
	AppSecret    string   `bson:"app_secret,omitempty" json:"app_secret,omitempty"`
	Domain       string   `bson:"domain,omitempty" json:"domain,omitempty"`
	MCPToken     string   `bson:"mcp_token,omitempty" json:"mcp_token,omitempty"`
	ReceiveID    string   `bson:"receive_id,omitempty" json:"receive_id,omitempty"`
	BotName      string   `bson:"bot_name,omitempty" json:"bot_name,omitempty"`
	BotOpenID    string   `bson:"bot_open_id,omitempty" json:"bot_open_id,omitempty"`
	AllowedUsers []string `bson:"allowed_users,omitempty" json:"allowed_users,omitempty"`
}

var legacyNoticeUnset = bson.M{
	"notice_url":               "",
	"notice_type":              "",
	"notice.channel":           "",
	"notice.notice_url":        "",
	"notice.notice_type":       "",
	"notice.feishu.enabled":    "",
	"notice.feishu.updated_at": "",
}

func userColl(ctx context.Context, cfg *config.AppConfig) (*mongo.Collection, func(), error) {
	if cfg == nil {
		return nil, func() {}, errors.New("usernotice: nil config")
	}
	uri := strings.TrimSpace(cfg.BotMongoURI)
	if uri == "" {
		return nil, func() {}, errors.New("usernotice: bot mongo not configured")
	}
	dbName := strings.TrimSpace(cfg.BotMongoDB)
	if dbName == "" {
		dbName = defaultBotMongoDB
	}
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, func() {}, err
	}
	return cli.Database(dbName).Collection("user"), func() { _ = cli.Disconnect(context.Background()) }, nil
}

// LoadCreds reads Feishu bot credentials from QT_DB.user.notice.feishu.
func LoadCreds(ctx context.Context, cfg *config.AppConfig, userID string) (*feishustore.Creds, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("usernotice: empty user id")
	}
	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}
	coll, disconnect, err := userColl(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer disconnect()

	var doc struct {
		Notice struct {
			Feishu FeishuSection `bson:"feishu"`
		} `bson:"notice"`
	}
	if err := coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return sectionToCreds(userID, doc.Notice.Feishu), nil
}

// ListCreds returns all users with Feishu credentials in Mongo.
func ListCreds(ctx context.Context, cfg *config.AppConfig) ([]feishustore.Creds, error) {
	coll, disconnect, err := userColl(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer disconnect()

	cur, err := coll.Find(ctx, bson.M{
		"notice.feishu.app_id":     bson.M{"$exists": true, "$ne": ""},
		"notice.feishu.app_secret": bson.M{"$exists": true, "$ne": ""},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := make([]feishustore.Creds, 0, 8)
	for cur.Next(ctx) {
		var doc struct {
			ID     primitive.ObjectID `bson:"_id"`
			Notice struct {
				Feishu FeishuSection `bson:"feishu"`
			} `bson:"notice"`
		}
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		c := sectionToCreds(doc.ID.Hex(), doc.Notice.Feishu)
		if c == nil || !c.Configured() {
			continue
		}
		out = append(out, *c)
	}
	return out, cur.Err()
}

// HasAppID reports whether another user already owns this Feishu app_id.
func HasAppID(ctx context.Context, cfg *config.AppConfig, appID, exceptUserID string) (bool, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return false, nil
	}
	coll, disconnect, err := userColl(ctx, cfg)
	if err != nil {
		return false, err
	}
	defer disconnect()

	filter := bson.M{
		"notice.feishu.app_id": appID,
	}
	if exceptUserID = strings.TrimSpace(exceptUserID); exceptUserID != "" {
		if oid, err := primitive.ObjectIDFromHex(exceptUserID); err == nil {
			filter["_id"] = bson.M{"$ne": oid}
		}
	}
	n, err := coll.CountDocuments(ctx, filter)
	return n > 0, err
}

// UpdateMCPToken patches notice.feishu.mcp_token for a user.
func UpdateMCPToken(ctx context.Context, cfg *config.AppConfig, userID, token string) error {
	userID = strings.TrimSpace(userID)
	token = strings.TrimSpace(token)
	if userID == "" || token == "" {
		return nil
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

	_, err = coll.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{
		"$set": bson.M{"notice.feishu.mcp_token": token},
	})
	return err
}

func sectionToCreds(userID string, s FeishuSection) *feishustore.Creds {
	appID := strings.TrimSpace(s.AppID)
	secret := strings.TrimSpace(s.AppSecret)
	if appID == "" || secret == "" {
		return nil
	}
	allowed := append([]string(nil), s.AllowedUsers...)
	receiveID := strings.TrimSpace(s.ReceiveID)
	if len(allowed) == 0 && receiveID != "" {
		allowed = []string{receiveID}
	}
	domain := strings.TrimSpace(s.Domain)
	if domain == "" {
		domain = "feishu"
	}
	return &feishustore.Creds{
		UserID:       userID,
		MCPToken:     strings.TrimSpace(s.MCPToken),
		AppID:        appID,
		AppSecret:    secret,
		Domain:       domain,
		BotName:      strings.TrimSpace(s.BotName),
		BotOpenID:    strings.TrimSpace(s.BotOpenID),
		AllowedUsers: allowed,
		HomeChannel:  receiveID,
		GroupPolicy:  "allowlist",
		Enabled:      s.Connected,
	}
}

func credsToSection(creds *feishustore.Creds) FeishuSection {
	section := FeishuSection{}
	if creds == nil {
		return section
	}
	section.Connected = creds.Configured() && creds.Enabled
	section.AppID = strings.TrimSpace(creds.AppID)
	section.AppSecret = strings.TrimSpace(creds.AppSecret)
	section.Domain = strings.TrimSpace(creds.Domain)
	if section.Domain == "" {
		section.Domain = "feishu"
	}
	section.MCPToken = strings.TrimSpace(creds.MCPToken)
	section.BotName = strings.TrimSpace(creds.BotName)
	section.BotOpenID = strings.TrimSpace(creds.BotOpenID)
	section.AllowedUsers = append([]string(nil), creds.AllowedUsers...)
	section.ReceiveID = resolveReceiveID(creds)
	if section.ReceiveID == "" {
		section.ReceiveID = strings.TrimSpace(creds.HomeChannel)
	}
	return section
}
