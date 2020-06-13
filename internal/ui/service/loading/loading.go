package loading

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

type Button struct {
	gtk.Button
	Spinner gtk.Spinner
}

func NewButton() *Button {
	s, _ := gtk.SpinnerNew()
	s.SetHAlign(gtk.ALIGN_CENTER)
	s.Start()
	s.Show()

	b, _ := gtk.ButtonNew()
	b.Add(s)
	b.SetSensitive(false) // unclickable
	b.Show()

	primitives.AddClass(b, "loading-button")

	return &Button{*b, *s}
}
