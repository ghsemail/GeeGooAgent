package exportmarkdown

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
)

// Export mirrors facts and episodes to MEMORY.md (Waku parity).
func Export(ctx context.Context, home, userID string, factStore *facts.PostgresStore, epStore *episodic.PostgresStore) error {
	if factStore == nil && epStore == nil {
		return nil
	}
	home = strings.TrimSpace(home)
	if home == "" {
		home = config.Home()
	}
	var factRows []facts.Row
	var eps []episodic.Episode
	if factStore != nil {
		rows, err := factStore.List(ctx, userID, 500)
		if err != nil {
			return err
		}
		factRows = rows
	}
	if epStore != nil {
		rows, err := epStore.List(ctx, userID, 500)
		if err != nil {
			return err
		}
		eps = rows
	}
	lines := []string{
		"# GeeGoo memory",
		"",
		"_A human-readable mirror of what the agent remembers. The source of truth is " +
			"PostgreSQL (`agent_facts` and `agent_episodes`, keyword-searchable via FTS); " +
			"this file is regenerated after every turn._",
		"",
		fmt.Sprintf("## Facts — semantic memory (%d)", len(factRows)),
		"",
	}
	if len(factRows) == 0 {
		lines = append(lines, "_none yet_")
	} else {
		for _, f := range factRows {
			lines = append(lines, fmt.Sprintf("- **%s** — %s", f.Subject, f.Content))
		}
	}
	lines = append(lines, "", fmt.Sprintf("## Episodes — episodic memory (%d)", len(eps)), "")
	if len(eps) == 0 {
		lines = append(lines, "_none yet_")
	} else {
		for _, e := range eps {
			lines = append(lines, fmt.Sprintf("- **%s** — %s", e.HappenedAt.Format("2006-01-02"), e.Summary))
		}
	}
	path := MemoryPath(home, userID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// MemoryPath returns the MEMORY.md path for a user (tenant-scoped when userID set).
func MemoryPath(home, userID string) string {
	userID = chatprompt.SanitizeUserID(userID)
	if userID == "" {
		return filepath.Join(home, "MEMORY.md")
	}
	return filepath.Join(home, "tenants", userID, "MEMORY.md")
}
