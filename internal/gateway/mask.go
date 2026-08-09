package gateway

import "strings"

// MaskMiddle keeps head/tail runes and replaces the middle with asterisks.
func MaskMiddle(s string, head, tail, starCount int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	n := len(runes)
	if starCount < 4 {
		starCount = 4
	}
	if n <= head+tail {
		return strings.Repeat("*", n)
	}
	return string(runes[:head]) + strings.Repeat("*", starCount) + string(runes[n-tail:])
}

// MaskAppID masks a Feishu App ID for dashboard display.
func MaskAppID(appID string) string {
	return MaskMiddle(appID, 4, 4, 6)
}

// MaskAppSecret masks an App Secret for dashboard display.
func MaskAppSecret(secret string) string {
	return MaskMiddle(secret, 3, 3, 8)
}
