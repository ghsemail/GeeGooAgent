package config

import (
	"os"
	"strings"
)

const (
	DefaultWeKnoraAPIURL     = "http://127.0.0.1:3481"
	DefaultWeKnoraWebURL     = "http://82.157.97.76:3480"
	DefaultWeKnoraKBID       = "3910ab1c-7f52-4a13-8a5a-5d3b99e75e36"
	DefaultWeKnoraAPIKeyFile = "/home/ubuntu/apps/WeKnora/.geegoo-bff-key"
)

// WeKnoraConfig is the JSON shape for the local WeKnora BFF.
type WeKnoraConfig struct {
	APIURL     string `json:"api_url,omitempty"`
	WebURL     string `json:"web_url,omitempty"`
	KBID       string `json:"kb_id,omitempty"`
	APIKey     string `json:"api_key,omitempty"`
	APIKeyFile string `json:"api_key_file,omitempty"`
}

// ResolvedWeKnora is a runtime-ready WeKnora connection.
type ResolvedWeKnora struct {
	APIURL     string
	WebURL     string
	KBID       string
	APIKey     string
	APIKeyFile string
}

// ResolvedWeKnora returns WeKnora endpoints and API key (env > config > key file).
func (c *AppConfig) ResolvedWeKnora() ResolvedWeKnora {
	out := ResolvedWeKnora{
		APIURL:     DefaultWeKnoraAPIURL,
		WebURL:     DefaultWeKnoraWebURL,
		KBID:       DefaultWeKnoraKBID,
		APIKeyFile: DefaultWeKnoraAPIKeyFile,
	}
	if c != nil {
		if v := strings.TrimSpace(c.WeKnora.APIURL); v != "" {
			out.APIURL = v
		}
		if v := strings.TrimSpace(c.WeKnora.WebURL); v != "" {
			out.WebURL = v
		}
		if v := strings.TrimSpace(c.WeKnora.KBID); v != "" {
			out.KBID = v
		}
		if v := strings.TrimSpace(c.WeKnora.APIKey); v != "" {
			out.APIKey = v
		}
		if v := strings.TrimSpace(c.WeKnora.APIKeyFile); v != "" {
			out.APIKeyFile = v
		}
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_WEKNORA_API_URL")); v != "" {
		out.APIURL = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_WEKNORA_WEB_URL")); v != "" {
		out.WebURL = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_WEKNORA_KB_ID")); v != "" {
		out.KBID = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_WEKNORA_API_KEY_FILE")); v != "" {
		out.APIKeyFile = v
	}
	if v := strings.TrimSpace(os.Getenv("GEEGOO_WEKNORA_API_KEY")); v != "" {
		out.APIKey = v
	}
	out.APIURL = trimSlash(out.APIURL)
	out.WebURL = trimSlash(out.WebURL)
	if out.APIKey == "" && out.APIKeyFile != "" {
		if raw, err := os.ReadFile(out.APIKeyFile); err == nil {
			out.APIKey = strings.TrimSpace(string(raw))
		}
	}
	return out
}
