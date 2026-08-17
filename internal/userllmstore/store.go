package userllmstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultBotMongoDB = "QT_DB"

var safeUserID = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Backend persists per-user gateway LLM prefs (Mongo via GeeGooBot service-api or direct Mongo).
type Backend struct {
	cfg       *config.AppConfig
	outputDir string

	mongoOnce sync.Once
	mongoCli  *mongo.Client
	mongoErr  error
}

// NewBackend creates a store. outputDir is used only when no DB/API backend is configured (local dev).
func NewBackend(cfg *config.AppConfig, outputDir string) *Backend {
	return &Backend{cfg: cfg, outputDir: strings.TrimSpace(outputDir)}
}

// Enabled reports whether settings are read/written via database (HTTP or Mongo), not local files.
func (b *Backend) Enabled() bool {
	if b == nil {
		return false
	}
	return b.serviceAPIEnabled() || b.mongoEnabled()
}

func (b *Backend) serviceAPIEnabled() bool {
	return strings.TrimSpace(b.serviceAPIURL()) != "" && strings.TrimSpace(b.serviceAPIKey()) != ""
}

func (b *Backend) mongoEnabled() bool {
	return strings.TrimSpace(b.mongoURI()) != ""
}

func (b *Backend) serviceAPIURL() string {
	if b == nil || b.cfg == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(b.cfg.BotServiceAPIURL), "/")
}

func (b *Backend) serviceAPIKey() string {
	if b == nil || b.cfg == nil {
		return ""
	}
	return strings.TrimSpace(b.cfg.BotServiceAPIKey)
}

func (b *Backend) mongoURI() string {
	if b == nil || b.cfg == nil {
		return ""
	}
	return strings.TrimSpace(b.cfg.BotMongoURI)
}

func (b *Backend) mongoDB() string {
	if b == nil || b.cfg == nil {
		return defaultBotMongoDB
	}
	if db := strings.TrimSpace(b.cfg.BotMongoDB); db != "" {
		return db
	}
	return defaultBotMongoDB
}

// Load returns user llm_settings from the configured backend. Nil when user has no stored prefs.
func (b *Backend) Load(ctx context.Context, userID string) (*Settings, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || b == nil {
		return nil, nil
	}
	if b.serviceAPIEnabled() {
		doc, err := b.loadHTTP(ctx, userID)
		if err != nil {
			return nil, err
		}
		return doc, nil
	}
	if b.mongoEnabled() {
		doc, err := b.loadMongo(ctx, userID)
		if err != nil {
			return nil, err
		}
		return doc, nil
	}
	return b.loadFile(userID)
}

// Save persists user llm_settings to the configured backend.
func (b *Backend) Save(ctx context.Context, userID string, doc *Settings) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || doc == nil || b == nil {
		return nil
	}
	doc.touchUpdatedAt()
	if b.serviceAPIEnabled() {
		return b.saveHTTP(ctx, userID, doc)
	}
	if b.mongoEnabled() {
		return b.saveMongo(ctx, userID, doc)
	}
	return b.saveFile(userID, doc)
}

func (b *Backend) loadHTTP(ctx context.Context, userID string) (*Settings, error) {
	body, _ := json.Marshal(map[string]string{"user_id": userID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.serviceAPIURL()+"/getUserLLMSettings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.serviceAPIKey())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getUserLLMSettings: %s", strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		LLMSettings *Settings `json:"llm_settings"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	return envelope.LLMSettings, nil
}

func (b *Backend) saveHTTP(ctx context.Context, userID string, doc *Settings) error {
	body, _ := json.Marshal(map[string]any{
		"user_id":      userID,
		"llm_settings": doc,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.serviceAPIURL()+"/setUserLLMSettings", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.serviceAPIKey())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("setUserLLMSettings: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}

func (b *Backend) mongoClient(ctx context.Context) (*mongo.Client, error) {
	if !b.mongoEnabled() {
		return nil, errors.New("bot mongo not configured")
	}
	b.mongoOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		b.mongoCli, b.mongoErr = mongo.Connect(ctx, options.Client().ApplyURI(b.mongoURI()))
	})
	if b.mongoErr != nil {
		return nil, b.mongoErr
	}
	if err := b.mongoCli.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return b.mongoCli, nil
}

func userObjectID(userID string) (primitive.ObjectID, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return primitive.NilObjectID, errors.New("empty user id")
	}
	return primitive.ObjectIDFromHex(userID)
}

func (b *Backend) loadMongo(ctx context.Context, userID string) (*Settings, error) {
	cli, err := b.mongoClient(ctx)
	if err != nil {
		return nil, err
	}
	oid, err := userObjectID(userID)
	if err != nil {
		return nil, err
	}
	var doc struct {
		LLMSettings *Settings `bson:"llm_settings"`
	}
	err = cli.Database(b.mongoDB()).Collection("user").FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return doc.LLMSettings, nil
}

func (b *Backend) saveMongo(ctx context.Context, userID string, doc *Settings) error {
	cli, err := b.mongoClient(ctx)
	if err != nil {
		return err
	}
	oid, err := userObjectID(userID)
	if err != nil {
		return err
	}
	_, err = cli.Database(b.mongoDB()).Collection("user").UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{"llm_settings": doc}},
	)
	return err
}

func (b *Backend) settingsPath(userID string) string {
	dir := b.outputDir
	if dir == "" {
		dir = "."
	}
	safe := safeUserID.ReplaceAllString(strings.TrimSpace(userID), "_")
	if safe == "" {
		safe = "anonymous"
	}
	return filepath.Join(dir, "user_llm_settings", safe+".json")
}

func (b *Backend) loadFile(userID string) (*Settings, error) {
	raw, err := os.ReadFile(b.settingsPath(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc Settings
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (b *Backend) saveFile(userID string, doc *Settings) error {
	path := b.settingsPath(userID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// SettingsFilePath returns the dev-only local JSON path for a user id.
func SettingsFilePath(outputDir, userID string) string {
	return NewBackend(nil, outputDir).settingsPath(userID)
}
