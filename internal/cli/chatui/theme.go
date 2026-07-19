package chatui

import (
	"github.com/charmbracelet/glamour"
)

// geegooMarkdownTheme — body 252, titles 214/220, meta 244.
const geegooMarkdownTheme = `{
  "document": { "color": "252", "margin": 0 },
  "paragraph": { "color": "252" },
  "block_quote": { "color": "244", "italic": true },
  "heading": { "bold": true, "color": "214" },
  "h1": { "bold": true, "color": "214" },
  "h2": { "bold": true, "color": "214" },
  "h3": { "bold": true, "color": "220" },
  "h4": { "bold": true, "color": "220" },
  "h5": { "color": "250" },
  "h6": { "color": "250" },
  "text": { "color": "252" },
  "strong": { "bold": true, "color": "220" },
  "em": { "italic": true, "color": "250" },
  "strikethrough": { "crossed_out": true, "color": "240" },
  "del": { "crossed_out": true, "color": "240" },
  "codespan": { "color": "214" },
  "code": { "color": "214" },
  "code_block": { "color": "214", "margin": 0 },
  "link": { "underline": true, "color": "220" },
  "link_text": { "underline": true, "color": "220" },
  "link_url": { "color": "244", "underline": true },
  "image": { "color": "220", "underline": true },
  "image_text": { "color": "240" },
  "list": { "color": "214" },
  "list_item": { "color": "252" },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "table": { "color": "252" },
  "table_header": { "bold": true, "color": "214" },
  "table_border": { "color": "238" },
  "hr": { "color": "240" }
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
