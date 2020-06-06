package ui

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/gotk3/gotk3/gtk"
)

type window struct {
	*gtk.Box
	Services    *service.View
	MessageView *messages.View
}

func newWindow() *window {
	services := service.NewView()
	services.SetSizeRequest(LeftWidth, -1)
	mesgview := messages.NewView()

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	separator.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(services, false, false, 0)
	box.PackStart(separator, false, false, 0)
	box.PackStart(mesgview, true, true, 0)
	box.Show()

	return &window{box, services, mesgview}
}
