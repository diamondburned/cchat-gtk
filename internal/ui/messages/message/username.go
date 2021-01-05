package message

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

var authorCSS = primitives.PrepareClassCSS("message-author", `
	.message-author {
		color: mix(@theme_bg_color, @theme_fg_color, 0.8);
	}
`)

func NewUsername() *labeluri.Label {
	user := labeluri.NewLabel(text.Rich{})
	user.SetXAlign(0) // left align
	user.SetVAlign(gtk.ALIGN_START)
	user.SetTrackVisitedLinks(false)
	user.Show()

	authorCSS(user)
	return user
}
