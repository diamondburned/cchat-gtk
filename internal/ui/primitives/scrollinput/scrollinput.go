package scrollinput

import "github.com/gotk3/gotk3/gtk"

func NewVScroll(maxHeight int) *gtk.ScrolledWindow {
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetProperty("propagate-natural-height", true)
	sw.SetProperty("max-content-height", maxHeight)

	return sw
}

func NewV(text *gtk.TextView, maxHeight int) *gtk.ScrolledWindow {
	// Wrap mode needed since we're not doing horizontal scrolling.
	text.SetWrapMode(gtk.WRAP_WORD_CHAR)

	sw := NewVScroll(maxHeight)
	sw.Add(text)

	return sw
}

func NewH(text *gtk.TextView) *gtk.ScrolledWindow {
	text.SetHExpand(true)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Add(text)
	sw.SetPolicy(gtk.POLICY_EXTERNAL, gtk.POLICY_NEVER)
	sw.SetProperty("propagate-natural-width", true)

	return sw
}
