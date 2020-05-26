package ui

import "github.com/gotk3/gotk3/gtk"

type header struct {
	*gtk.Box
}

func newHeader() *header {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	// TODO
	return &header{box}
}
