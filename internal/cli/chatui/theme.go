package chatui

import (
	"github.com/charmbracelet/glamour"
)

// geegooMarkdownTheme mirrors Python _GEEGOO_CHAT_THEME (white body, yellow accents).
// ANSI-256 indices avoid glamour/chroma defaults that use magenta (e.g. link_text "35").
const geegooMarkdownTheme = `{
  "document": { "color": "252", "margin": 0 },
  "paragraph": { "color": "252" },
  "block_quote": { "color": "214", "italic": true },
  "heading": { "bold": true, "color": "214" },
  "h1": { "bold": true, "color": "214" },
  "h2": { "bold": true, "color": "214" },
  "h3": { "bold": true, "color": "220" },
  "h4": { "color": "220" },
  "h5": { "color": "220" },
  "h6": { "color": "220" },
  "text": { "color": "252" },
  "strong": { "bold": true, "color": "220" },
  "em": { "italic": true, "color": "252" },
  "strikethrough": { "crossed_out": true, "color": "245" },
  "del": { "crossed_out": true, "color": "245" },
  "codespan": { "bold": true, "color": "214" },
  "code": { "color": "214" },
  "code_block": { "color": "214", "margin": 0 },
  "link": { "underline": true, "color": "220" },
  "link_text": { "underline": true, "color": "220" },
  "link_url": { "underline": true, "color": "214" },
  "image": { "color": "220", "underline": true },
  "image_text": { "color": "245" },
  "list": { "color": "220" },
  "list_item": { "color": "252" },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "table": { "color": "252" },
  "table_header": { "bold": true, "color": "220" },
  "table_border": { "color": "214" },
  "hr": { "color": "245" }
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
