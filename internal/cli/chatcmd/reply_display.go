package chatcmd

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// ReplyDisplayResult is the outcome of /stream or /reply commands.
type ReplyDisplayResult struct {
	OK      bool
	Persist bool
	Message string
	Display *config.DisplayConfig
}

// ApplyStream toggles stream_reply (on|off).
func ApplyStream(display *config.DisplayConfig, args []string) ReplyDisplayResult {
	if display == nil {
		return ReplyDisplayResult{OK: false, Message: "display config is nil"}
	}
	d := *display
	d.Normalize()
	if len(args) == 0 {
		return ReplyDisplayResult{
			OK: true, Display: &d,
			Message: formatReplyDisplayStatus(d),
		}
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "on", "true", "1", "yes":
		v := true
		d.StreamReply = &v
	case "off", "false", "0", "no":
		v := false
		d.StreamReply = &v
	default:
		return ReplyDisplayResult{OK: false, Display: &d, Message: "usage: /stream [on|off]"}
	}
	d.Normalize()
	return ReplyDisplayResult{
		OK: true, Persist: true, Display: &d,
		Message: "stream_reply=" + fmt.Sprintf("%v", d.StreamReplyEnabled()),
	}
}

// ApplyReplyFormat sets reply_format (markdown|plain).
func ApplyReplyFormat(display *config.DisplayConfig, args []string) ReplyDisplayResult {
	if display == nil {
		return ReplyDisplayResult{OK: false, Message: "display config is nil"}
	}
	d := *display
	d.Normalize()
	if len(args) == 0 {
		return ReplyDisplayResult{
			OK: true, Display: &d,
			Message: formatReplyDisplayStatus(d),
		}
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "markdown", "md":
		d.ReplyFormat = config.ReplyFormatMarkdown
	case "plain", "text":
		d.ReplyFormat = config.ReplyFormatPlain
	default:
		return ReplyDisplayResult{OK: false, Display: &d, Message: "usage: /reply [markdown|plain]"}
	}
	d.Normalize()
	return ReplyDisplayResult{
		OK: true, Persist: true, Display: &d,
		Message: "reply_format=" + d.ReplyFormat,
	}
}

func formatReplyDisplayStatus(d config.DisplayConfig) string {
	stream := "off"
	if d.StreamReplyEnabled() {
		stream = "on"
	}
	return "stream_reply=" + stream + " reply_format=" + d.ReplyFormat
}
