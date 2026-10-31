package runtime

import (
	"fmt"
	"strings"
)

// LockedSkillTurnHint is LLM-visible guidance when Monday locks one skill for this turn.
func LockedSkillTurnHint(lockedSkill string) string {
	id := strings.TrimSpace(lockedSkill)
	if id == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Monday locked skill (this turn)\n")
	fmt.Fprintf(&b, "- **locked_skill**: `%s` — load this skill with **skill_view** and follow its SKILL.md; do not switch to other skills mid-diagnosis unless the user clearly changes topic.\n", id)
	return strings.TrimSpace(b.String())
}
