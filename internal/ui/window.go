package ui

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/gotk3/gotk3/gtk"
)

type window struct {
	*gtk.Paned
	Services    *service.View
	MessageView *messages.View
}

func newWindow() *window {
	services := service.NewView()
	services.SetSizeRequest(leftMinWidth, -1)
	services.Show()

	mesgview := messages.NewView()
	mesgview.Show()

	pane, _ := gtk.PanedNew(gtk.ORIENTATION_HORIZONTAL)
	pane.Pack1(services, false, false)
	pane.Pack2(mesgview, true, false)
	pane.SetPosition(leftCurrentWidth)
	pane.Show()

	return &window{pane, services, mesgview}
}
