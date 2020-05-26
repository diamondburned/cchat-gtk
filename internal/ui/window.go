package ui

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/gotk3/gotk3/gtk"
)

type window struct {
	*gtk.Box
	Services    *service.View
	MessageView *message.View
}

func newWindow() *window {
	services := service.NewView()
	services.SetSizeRequest(LeftWidth, -1)
	mesgview := message.NewView()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()
	box.PackStart(services, false, false, 0)
	box.PackStart(mesgview, true, true, 0)

	return &window{box, services, mesgview}
}
