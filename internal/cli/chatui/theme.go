package chatui

import (
	"github.com/charmbracelet/glamour"
)

// geegooMarkdownTheme mirrors Python _GEEGOO_CHAT_THEME (gold finance palette).
const geegooMarkdownTheme = `{
  "document": { "color": "#E5E7EB", "margin": 0 },
  "paragraph": { "color": "#E5E7EB" },
  "block_quote": { "color": "#FFBF00", "italic": true },
  "heading": { "bold": true, "color": "#FFBF00" },
  "h1": { "bold": true, "color": "#FFBF00" },
  "h2": { "bold": true, "color": "#FFBF00" },
  "h3": { "bold": true, "color": "#FFD700" },
  "h4": { "color": "#FFD700" },
  "h5": { "color": "#FFD700" },
  "h6": { "color": "#FFD700" },
  "text": { "color": "#E5E7EB" },
  "strong": { "bold": true, "color": "#FFD700" },
  "em": { "italic": true, "color": "#E5E7EB" },
  "codespan": { "bold": true, "color": "#FFBF00" },
  "code": { "color": "#FFBF00" },
  "code_block": { "color": "#FFBF00" },
  "link": { "underline": true, "color": "#FFD700" },
  "link_text": { "underline": true, "color": "#FFD700" },
  "link_url": { "underline": true, "color": "#FFBF00" },
  "list": { "color": "#FFD700" },
  "list_item": { "color": "#E5E7EB" },
  "table": { "color": "#E5E7EB" },
  "table_header": { "bold": true, "color": "#FFD700" },
  "table_border": { "color": "#FFBF00" },
  "hr": { "color": "#9CA3AF" }
}`

func newMarkdownRenderer(width int) (*glamour.TermRenderer, error) {
	if width < 40 {
		width = 40
	}
	return glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(geegooMarkdownTheme)),
		glamour.WithWordWrap(width),
	)
}
