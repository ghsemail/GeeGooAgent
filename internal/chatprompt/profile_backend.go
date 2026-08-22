package chatprompt

import (
	"context"
	"strings"
)

// ProfileBackend loads/saves context profile content (scoped DB with optional file fallback).
type ProfileBackend interface {
	Get(ctx context.Context, userID string, ref ProfileRef) (content string, ok bool)
	Put(ctx context.Context, userID string, ref ProfileRef, content string) error
}

var profileBackend ProfileBackend

// SetProfileBackend wires scoped preference storage (PostgreSQL). Nil keeps file-only mode.
func SetProfileBackend(b ProfileBackend) {
	profileBackend = b
}

func loadProfileMerged(home, userID string, ref ProfileRef) (LoadedProfile, bool) {
	path := AgentsPathForRef(home, userID, ref)
	out := LoadedProfile{Ref: ref, Path: path}
	if profileBackend != nil {
		content, ok := profileBackend.Get(context.Background(), userID, ref)
		if ok && strings.TrimSpace(content) != "" {
			out.Content = strings.TrimRight(strings.TrimSpace(content), "\n") + "\n"
			out.Bytes = len([]byte(out.Content))
			return out, true
		}
	}
	return loadProfileFile(home, userID, ref)
}

func saveProfileMerged(home, userID string, ref ProfileRef, content string) error {
	if profileBackend != nil {
		return profileBackend.Put(context.Background(), userID, ref, content)
	}
	return saveProfileFile(home, userID, ref, content)
}
