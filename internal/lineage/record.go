package lineage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Record is one compression/hygiene event in session bloodline.
type Record struct {
	Generation   int    `json:"generation"`
	ParentID     string `json:"parent_id,omitempty"`
	Hygiene      bool   `json:"hygiene,omitempty"`
	MsgsBefore   int    `json:"msgs_before"`
	MsgsAfter    int    `json:"msgs_after"`
	TokensBefore int    `json:"tokens_before"`
	TokensAfter  int    `json:"tokens_after"`
	SummaryHash  string `json:"summary_hash,omitempty"`
	SummaryChars int    `json:"summary_chars"`
}

// SummaryHashPrefix returns a short stable hash for compaction summary text.
func SummaryHashPrefix(summary string) string {
	if summary == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(summary))
	return hex.EncodeToString(sum[:4])
}

// FormatLine renders one record for CLI output.
func (r Record) FormatLine() string {
	kind := "compress"
	if r.Hygiene {
		kind = "hygiene"
	}
	return fmt.Sprintf("g%d %s parent=%s msgs %d→%d tokens %d→%d summary#%s (%d chars)",
		r.Generation, kind, r.ParentID, r.MsgsBefore, r.MsgsAfter,
		r.TokensBefore, r.TokensAfter, r.SummaryHash, r.SummaryChars)
}
