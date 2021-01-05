package message

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var timestampCSS = primitives.PrepareClassCSS("message-time", `
	.message-time {
		opacity: 0.3;
		font-size: 0.8em;
		margin-top: 0.2em;
		margin-bottom: 0.2em;
	}
`)

func NewTimestamp() *gtk.Label {
	ts, _ := gtk.LabelNew("")
	ts.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
	ts.SetXAlign(0.5) // centre align
	ts.SetVAlign(gtk.ALIGN_END)
	ts.Show()

	timestampCSS(ts)
	return ts
}
