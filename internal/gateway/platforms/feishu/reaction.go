package feishu

import (
	"context"
	"fmt"
	"log/slog"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

const (
	reactionInProgress = "Typing"
	reactionFailure    = "CrossMark"
	reactionCacheMax   = 1024
)

// MarkProcessing adds a Typing reaction to the user message (Hermes-aligned).
func (a *Adapter) MarkProcessing(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}
	a.ensureAPI()
	rid, err := a.addReaction(ctx, messageID, reactionInProgress)
	if err != nil {
		slog.Warn("feishu: mark processing reaction failed", "err", err, "message_id", messageID)
		return nil // never block the agent turn
	}
	if rid != "" {
		a.storeReaction(messageID, rid)
	}
	return nil
}

// ClearProcessing removes the Typing reaction after a successful reply.
func (a *Adapter) ClearProcessing(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}
	rid := a.takeReaction(messageID)
	if rid == "" {
		return nil
	}
	a.ensureAPI()
	if err := a.removeReaction(ctx, messageID, rid); err != nil {
		slog.Warn("feishu: clear processing reaction failed", "err", err, "message_id", messageID)
	}
	return nil
}

// MarkFailed swaps Typing for CrossMark when the agent turn fails.
func (a *Adapter) MarkFailed(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}
	a.ensureAPI()
	if rid := a.takeReaction(messageID); rid != "" {
		_ = a.removeReaction(ctx, messageID, rid)
	}
	if _, err := a.addReaction(ctx, messageID, reactionFailure); err != nil {
		slog.Warn("feishu: mark failed reaction failed", "err", err, "message_id", messageID)
	}
	return nil
}

func (a *Adapter) addReaction(ctx context.Context, messageID, emoji string) (string, error) {
	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().EmojiType(emoji).Build()).
			Build()).
		Build()
	resp, err := a.api.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu reaction create: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data != nil && resp.Data.ReactionId != nil {
		return *resp.Data.ReactionId, nil
	}
	return "", nil
}

func (a *Adapter) removeReaction(ctx context.Context, messageID, reactionID string) error {
	req := larkim.NewDeleteMessageReactionReqBuilder().
		MessageId(messageID).
		ReactionId(reactionID).
		Build()
	resp, err := a.api.Im.V1.MessageReaction.Delete(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("feishu reaction delete: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func (a *Adapter) storeReaction(messageID, reactionID string) {
	a.reactMu.Lock()
	defer a.reactMu.Unlock()
	if a.pendingReactions == nil {
		a.pendingReactions = make(map[string]string)
	}
	a.pendingReactions[messageID] = reactionID
	for len(a.pendingReactions) > reactionCacheMax {
		for k := range a.pendingReactions {
			delete(a.pendingReactions, k)
			break
		}
	}
}

func (a *Adapter) takeReaction(messageID string) string {
	a.reactMu.Lock()
	defer a.reactMu.Unlock()
	if a.pendingReactions == nil {
		return ""
	}
	rid := a.pendingReactions[messageID]
	delete(a.pendingReactions, messageID)
	return rid
}