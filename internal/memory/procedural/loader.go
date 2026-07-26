// Package procedural implements procedural memory — SKILL.md files loaded when relevant.
package procedural

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Skill is one parsed SKILL.md (Anthropic frontmatter or plain markdown).
type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
}

// Loader scans skill directories and matches skills to user messages.
type Loader struct {
	dirs      []string
	skills    []Skill
	signature []fileSig
}

// Dirs returns configured scan roots (repo skills/, workspace skills/).
func (l *Loader) Dirs() []string {
	if l == nil {
		return nil
	}
	out := make([]string, len(l.dirs))
	copy(out, l.dirs)
	return out
}

type fileSig struct {
	path string
	mod  int64
}

var (
	frontmatterRE = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)
	wordRE        = regexp.MustCompile(`[a-z0-9]{2,}`)
)

// NewLoader creates a loader over the given directories (repo skills/, workspace skills/).
func NewLoader(dirs ...string) *Loader {
	l := &Loader{dirs: dirs}
	l.refresh()
	return l
}

// List returns all loaded skills.
func (l *Loader) List() []Skill {
	if l == nil {
		return nil
	}
	l.maybeRefresh()
	out := make([]Skill, len(l.skills))
	copy(out, l.skills)
	return out
}

// Match returns up to maxSkills skills whose keywords overlap the message.
func (l *Loader) Match(message string, maxSkills int) []Skill {
	if l == nil || strings.TrimSpace(message) == "" {
		return nil
	}
	l.maybeRefresh()
	if maxSkills <= 0 {
		maxSkills = 2
	}
	msgWords := tokenize(message)
	lowerMsg := strings.ToLower(message)
	type scored struct {
		score int
		skill Skill
	}
	var hits []scored
	for _, sk := range l.skills {
		if !InjectInChat(ClassifyPath(sk.Path)) {
			continue
		}
		score := overlapScore(msgWords, sk.Name+" "+sk.Description)
		if score < 2 {
			if strings.Contains(lowerMsg, strings.ToLower(sk.Name)) {
				score = 2
			}
		}
		if score >= 2 {
			hits = append(hits, scored{score: score, skill: sk})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > maxSkills {
		hits = hits[:maxSkills]
	}
	out := make([]Skill, len(hits))
	for i, h := range hits {
		out[i] = h.skill
	}
	return out
}

// Format injects matched skill bodies for the prompt.
func Format(matched []Skill) string {
	if len(matched) == 0 {
		return ""
	}
	var b strings.Builder
	for i, sk := range matched {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("### ")
		b.WriteString(sk.Name)
		b.WriteString("\n")
		body := strings.TrimSpace(sk.Body)
		if body == "" {
			body = sk.Description
		}
		b.WriteString(body)
	}
	return b.String()
}

// Refresh rescans skill directories (Waku create_skill parity).
func (l *Loader) Refresh() {
	if l != nil {
		l.refresh()
	}
}

func (l *Loader) maybeRefresh() {
	sig := l.scanSignature()
	if stringSig(sig) != stringSig(l.signature) {
		l.refresh()
	}
}

func (l *Loader) refresh() {
	l.skills = nil
	l.signature = l.scanSignature()
	for _, dir := range l.dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if d.Name() != "SKILL.md" {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if sk := parseSkill(string(raw), path); sk != nil {
				l.skills = append(l.skills, *sk)
			}
			return nil
		})
	}
	sort.Slice(l.skills, func(i, j int) bool { return l.skills[i].Name < l.skills[j].Name })
}

func (l *Loader) scanSignature() []fileSig {
	var sig []fileSig
	for _, dir := range l.dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			sig = append(sig, fileSig{path: path, mod: info.ModTime().UnixNano()})
			return nil
		})
	}
	sort.Slice(sig, func(i, j int) bool { return sig[i].path < sig[j].path })
	return sig
}

func stringSig(sig []fileSig) string {
	var parts []string
	for _, s := range sig {
		parts = append(parts, s.path+":"+strconv.FormatInt(s.mod, 10))
	}
	return strings.Join(parts, "|")
}

// ParseSkillText validates SKILL.md frontmatter + body without reading from disk.
func ParseSkillText(text string) *Skill {
	return parseSkill(text, "inline")
}

func parseSkill(text, path string) *Skill {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if m := frontmatterRE.FindStringSubmatch(text); len(m) == 3 {
		fields := parseFrontmatter(m[1])
		name := fields["name"]
		desc := fields["description"]
		if name == "" {
			name = filepath.Base(filepath.Dir(path))
		}
		return &Skill{Name: name, Description: desc, Body: strings.TrimSpace(m[2]), Path: path}
	}
	name := filepath.Base(filepath.Dir(path))
	lines := strings.Split(text, "\n")
	desc := ""
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimLeft(line, "# "))
		if line != "" {
			desc = line
			break
		}
	}
	return &Skill{Name: name, Description: desc, Body: text, Path: path}
}

func parseFrontmatter(block string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(block, "\n") {
		if idx := strings.Index(line, ":"); idx > 0 {
			k := strings.TrimSpace(line[:idx])
			v := strings.Trim(strings.TrimSpace(line[idx+1:]), `"'`)
			out[k] = v
		}
	}
	return out
}

func tokenize(text string) map[string]struct{} {
	words := map[string]struct{}{}
	lower := strings.ToLower(text)
	for _, w := range wordRE.FindAllString(lower, -1) {
		words[w] = struct{}{}
	}
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		if !unicode.Is(unicode.Han, runes[i]) {
			continue
		}
		words[string(runes[i])] = struct{}{}
		if i+1 < len(runes) && unicode.Is(unicode.Han, runes[i+1]) {
			words[string(runes[i : i+2])] = struct{}{}
		}
	}
	return words
}

func overlapScore(msgWords map[string]struct{}, skillText string) int {
	skillWords := tokenize(skillText)
	n := 0
	for w := range msgWords {
		if _, ok := skillWords[w]; ok {
			n++
		}
	}
	return n
}
