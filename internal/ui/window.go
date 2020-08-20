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

type Controller interface {
	service.Controller
	messages.Controller
}

func newWindow(mainctl Controller) *window {
	services := service.NewView(mainctl)
	services.SetSizeRequest(leftMinWidth, -1)
	services.Show()

	mesgview := messages.NewView(mainctl)
	mesgview.Show()

	pane, _ := gtk.PanedNew(gtk.ORIENTATION_HORIZONTAL)
	pane.Pack1(services, false, false)
	pane.Pack2(mesgview, true, false)
	pane.SetPosition(leftCurrentWidth)
	pane.Show()

	return &window{pane, services, mesgview}
}

func (w *window) AllServices() []*service.Service {
	return w.Services.Services.Services
}
