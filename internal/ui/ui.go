package ui

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
)

const LeftWidth = 220

type Application struct {
	window *window
	header *header
}

var (
	_ gts.Windower         = (*Application)(nil)
	_ gts.Headerer         = (*Application)(nil)
	_ server.RowController = (*Application)(nil)
)

func NewApplication() *Application {
	app := &Application{
		window: newWindow(),
		header: newHeader(),
	}

	return app
}

func (app *Application) AddService(svc cchat.Service) {
	app.window.Services.AddService(svc, app)
}

func (app *Application) MessageRowSelected(_ *server.Row, smsg cchat.ServerMessage) {
	app.window.MessageView.JoinServer(smsg)
}

func (app *Application) Header() gtk.IWidget {
	return app.header
}

func (app *Application) Window() gtk.IWidget {
	return app.window
}
