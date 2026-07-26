package procedural

import (
	"path/filepath"
	"strings"
)

// ExtensionSkillRel is the canonical extensions directory under skills/.
const ExtensionSkillRel = "skills/extensions"

// LegacyExtensionSkillRel is the pre-P0 bundled path (still scanned for scripts).
const LegacyExtensionSkillRel = "skills/bundled"

// IsExtensionPath reports whether path is under extensions/ or legacy bundled/.
func IsExtensionPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(p, "/extensions/") || strings.Contains(p, "/bundled/")
}

// ExtensionScriptCandidates returns script paths to try for an extension skill.
func ExtensionScriptCandidates(projectRoot, skillName, scriptFile string) []string {
	if scriptFile == "" {
		scriptFile = "web_search.py"
	}
	var out []string
	if root := strings.TrimSpace(projectRoot); root != "" {
		for _, rel := range []string{ExtensionSkillRel, LegacyExtensionSkillRel} {
			out = append(out, filepath.Join(root, rel, skillName, "scripts", scriptFile))
		}
	}
	return out
}
