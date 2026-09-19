package diagram

import (
	"strings"
	"unicode"
)

// Slug turns a step/phase name into an Archify id: ^[a-zA-Z][a-zA-Z0-9_-]*$
func Slug(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "node"
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) && r < 128:
			b.WriteRune(r)
			lastUnderscore = false
		case unicode.IsDigit(r) || r == '_' || r == '-':
			if b.Len() == 0 && !unicode.IsDigit(r) {
				continue
			}
			b.WriteRune(r)
			lastUnderscore = r == '_'
		default:
			if b.Len() > 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" {
		return "node"
	}
	if c := out[0]; c < 'A' || c > 'z' || (c > 'Z' && c < 'a') {
		return "n" + out
	}
	return out
}

func uniqueID(used map[string]int, base string) string {
	base = Slug(base)
	n := used[base]
	used[base] = n + 1
	if n == 0 {
		return base
	}
	return base + "_" + itoa(n+1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
