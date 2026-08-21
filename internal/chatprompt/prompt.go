package chatprompt

import "github.com/ghsemail/GeeGooAgent/internal/config"

// System returns the default geegoo chat system prompt.
func System() string {
	return DefaultBuilder().Build()
}

// SystemForUser builds the system prompt with tenant-scoped SOUL and context profiles.
func SystemForUser(userID string) string {
	return SystemForUserProfiles(config.Home(), userID, nil, DefaultProfileLimits())
}

// SystemForSession builds system prompt including session-bound context profiles.
func SystemForSession(userID string, sessionRefs []string, limits ProfileLimits) string {
	return SystemForUserProfiles(config.Home(), userID, sessionRefs, limits)
}
