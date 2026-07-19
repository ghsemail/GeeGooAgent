package chattui

import "strings"

const minViewportHeight = 4

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// showWelcomeBanner reports whether the Hermes welcome panel should be shown.
func (m *Model) showWelcomeBanner() bool {
	if m.banner == "" {
		return false
	}
	s := m.activeSlot()
	if s == nil {
		return true
	}
	for _, block := range s.Blocks {
		if block.Kind == KindUser {
			return false
		}
	}
	return true
}

// canFixWelcomeBanner reports whether the banner fits above the scrollable viewport.
func (m Model) canFixWelcomeBanner() bool {
	if !m.showWelcomeBanner() {
		return false
	}
	if m.height <= 0 {
		return true
	}
	footer := m.footerLineCount()
	return lineCount(m.banner) <= m.height-footer-minViewportHeight
}

func (m Model) fixedWelcomeBannerLines() int {
	if !m.canFixWelcomeBanner() {
		return 0
	}
	return lineCount(m.banner)
}
