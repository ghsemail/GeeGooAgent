package chatui

// GeeGoo terminal palette — warm gold brand on neutral gray scale.
// Terminals cannot change point size; hierarchy uses weight, italic, and brightness.
//
// Tier 0  brand   220 bold   rules, ⚕, logo, focus
// Tier 1  title   214 bold   section heads, clarify ?, banner labels
// Tier 2  body    252        user text, tool lines, assistant plain body
// Tier 3  user    250        committed user prompt (slightly softer than reply)
// Tier 4  meta    244        chevrons, stats, footers, hints
// Tier 5  whisper 240        soft dividers, placeholders, panel chrome
const (
	ColorGold   = "220" // brand gold
	ColorAmber  = "214" // title amber
	ColorAccent = "220"
	ColorText     = "252" // primary body
	ColorUser     = "250" // user message body
	ColorThinking = "244" // thinking / italic secondary
	ColorDim      = "244" // meta (alias)
	ColorWhisper  = "240" // faint chrome
	ColorBorder   = "238" // input / panel borders
	ColorErr      = "203"
	ColorOK       = "114"
	ColorRunning  = "120"
	ColorBgPrompt = "237" // user prompt band
)

const (
	colorGold     = ColorGold
	colorAmber    = ColorAmber
	colorAccent   = ColorAccent
	colorText     = ColorText
	colorUser     = ColorUser
	colorThinking = ColorThinking
	colorDim      = ColorDim
	colorWhisper  = ColorWhisper
	colorBorder   = ColorBorder
	colorErr      = ColorErr
	colorOK       = ColorOK
	colorRunning  = ColorRunning
	colorBgPrompt = ColorBgPrompt
)
