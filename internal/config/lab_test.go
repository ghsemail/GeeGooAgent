package config

import (
	"os"
	"testing"
)

func TestLabEnabledDefaultOff(t *testing.T) {
	_ = os.Unsetenv("GEEGOO_AGENT_LAB_ENABLED")
	c := &AppConfig{}
	if c.LabEnabled() {
		t.Fatal("expected lab off by default")
	}
}

func TestLabEnabledEnv(t *testing.T) {
	t.Setenv("GEEGOO_AGENT_LAB_ENABLED", "true")
	c := &AppConfig{}
	if !c.LabEnabled() {
		t.Fatal("expected lab on from env")
	}
}

func TestLabAPIURLDefault(t *testing.T) {
	_ = os.Unsetenv("GEEGOO_AGENT_LAB_API_URL")
	c := &AppConfig{}
	if c.LabAPIURL() != DefaultSignalLabAPIURL {
		t.Fatalf("url=%q", c.LabAPIURL())
	}
}
