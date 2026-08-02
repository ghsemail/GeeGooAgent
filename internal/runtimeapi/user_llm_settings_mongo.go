package runtimeapi

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultBotMongoDB = "QT_DB"

var (
	userLLMMongoOnce sync.Once
	userLLMMongoCli  *mongo.Client
	userLLMMongoErr  error
)

func (h *Handler) botMongoURI() string {
	if h.App == nil || h.App.Config == nil {
		return ""
	}
	return strings.TrimSpace(h.App.Config.BotMongoURI)
}

func (h *Handler) botMongoDB() string {
	if h.App == nil || h.App.Config == nil {
		return defaultBotMongoDB
	}
	if db := strings.TrimSpace(h.App.Config.BotMongoDB); db != "" {
		return db
	}
	return defaultBotMongoDB
}

func (h *Handler) userLLMMongoEnabled() bool {
	return h.botMongoURI() != ""
}

func (h *Handler) userLLMMongoClient(ctx context.Context) (*mongo.Client, error) {
	if !h.userLLMMongoEnabled() {
		return nil, errors.New("bot mongo not configured")
	}
	userLLMMongoOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		userLLMMongoCli, userLLMMongoErr = mongo.Connect(ctx, options.Client().ApplyURI(h.botMongoURI()))
	})
	if userLLMMongoErr != nil {
		return nil, userLLMMongoErr
	}
	if err := userLLMMongoCli.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return userLLMMongoCli, nil
}

func userObjectID(userID string) (primitive.ObjectID, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return primitive.NilObjectID, errors.New("empty user id")
	}
	return primitive.ObjectIDFromHex(userID)
}

func (h *Handler) loadUserLLMSettingsMongo(ctx context.Context, userID string) (*userLLMSettings, error) {
	cli, err := h.userLLMMongoClient(ctx)
	if err != nil {
		return nil, err
	}
	oid, err := userObjectID(userID)
	if err != nil {
		return nil, err
	}
	var doc struct {
		LLMSettings *userLLMSettings `bson:"llm_settings"`
	}
	err = cli.Database(h.botMongoDB()).Collection("user").FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	if doc.LLMSettings == nil {
		return nil, nil
	}
	return doc.LLMSettings, nil
}

func (h *Handler) saveUserLLMSettingsMongo(ctx context.Context, userID string, settings *userLLMSettings) error {
	if settings == nil {
		return nil
	}
	cli, err := h.userLLMMongoClient(ctx)
	if err != nil {
		return err
	}
	oid, err := userObjectID(userID)
	if err != nil {
		return err
	}
	settings.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err = cli.Database(h.botMongoDB()).Collection("user").UpdateOne(
		ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{"llm_settings": settings}},
	)
	return err
}

func (h *Handler) loadUserLLMSettings(userID string) (*userLLMSettings, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	if h.botServiceAPIEnabled() {
		doc, err := h.loadUserLLMSettingsHTTP(ctx, userID)
		if err == nil {
			if doc != nil {
				return doc, nil
			}
			fileDoc, fileErr := loadUserLLMSettings(h.userSettingsOutputDir(), userID)
			if fileDoc != nil {
				_ = h.saveUserLLMSettings(userID, fileDoc)
				return fileDoc, nil
			}
			return nil, fileErr
		}
	}

	if h.userLLMMongoEnabled() {
		doc, err := h.loadUserLLMSettingsMongo(ctx, userID)
		if err == nil && doc != nil {
			return doc, nil
		}
		if err == nil && doc == nil {
			fileDoc, fileErr := loadUserLLMSettings(h.userSettingsOutputDir(), userID)
			if fileDoc != nil {
				_ = h.saveUserLLMSettingsMongo(ctx, userID, fileDoc)
				return fileDoc, nil
			}
			return nil, fileErr
		}
	}
	return loadUserLLMSettings(h.userSettingsOutputDir(), userID)
}

func (h *Handler) saveUserLLMSettings(userID string, doc *userLLMSettings) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || doc == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	if h.botServiceAPIEnabled() {
		if err := h.saveUserLLMSettingsHTTP(ctx, userID, doc); err == nil {
			return nil
		}
	}

	if h.userLLMMongoEnabled() {
		if err := h.saveUserLLMSettingsMongo(ctx, userID, doc); err == nil {
			return nil
		}
	}
	return saveUserLLMSettings(h.userSettingsOutputDir(), userID, doc)
}
