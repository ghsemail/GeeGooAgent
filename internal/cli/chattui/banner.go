package chattui

import (
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatui"
)

func bannerOptsFromRepl(r *chatrepl.Repl) chatui.BannerOptions {
	if r == nil {
		return chatui.BannerOptions{}
	}
	return r.BannerOptions()
}
