package chatui

// Hermes / Claude Code inspired palette (matches Python chat_ui.py + chat_banner.py).
// ANSI-256 indices are used for lipgloss so SSH / Windows terminals avoid violet fallbacks.
const (
	ColorGold   = "220" // yellow
	ColorAmber  = "214" // light gold
	ColorAccent = "220"
	ColorText     = "252" // white / light gray
	ColorThinking = "252"
	ColorDim      = "245" // bright gray
	ColorBorder   = "214"
	ColorErr      = "203"
	ColorOK       = "114"
	ColorRunning  = "120" // light green — in-progress elapsed
	ColorBgPrompt = "236" // dark gray user prompt band (Grok-style)
)

// Back-compat aliases for chatui internals.
const (
	colorGold     = ColorGold
	colorAmber    = ColorAmber
	colorAccent   = ColorAccent
	colorText     = ColorText
	colorThinking = ColorThinking
	colorDim      = ColorDim
	colorBorder   = ColorBorder
	colorErr      = ColorErr
	colorOK       = ColorOK
	colorRunning  = ColorRunning
	colorBgPrompt = ColorBgPrompt
)
