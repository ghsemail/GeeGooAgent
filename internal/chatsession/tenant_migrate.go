package chatsession

import "strings"

// StampOrphanSessions assigns userID to sessions with no owner in metadata/index.
func StampOrphanSessions(store SessionStore, userID string) (stamped int, err error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || store == nil {
		return 0, nil
	}
	entries, err := store.ListIndexedSessions()
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if UserIDFromEntry(e) != "" {
			continue
		}
		s, loadErr := store.Load(e.ID)
		if loadErr != nil || s == nil {
			continue
		}
		SetUserID(s, userID)
		if saveErr := store.Save(s); saveErr != nil {
			return stamped, saveErr
		}
		stamped++
	}
	return stamped, nil
}
