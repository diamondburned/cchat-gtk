package ui

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/auth"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
	"github.com/markbates/pkger"
)

func init() {
	// Load the local CSS.
	gts.LoadCSS(pkger.Include("/internal/ui/style.css"))
}

const LeftWidth = 220

type App struct {
	window *window
	header *header

	// used to keep track of what row to highlight and unhighlight
	lastRowHighlighter func(bool)
}

var (
	_ gts.Windower       = (*App)(nil)
	_ gts.Headerer       = (*App)(nil)
	_ service.Controller = (*App)(nil)
)

func NewApplication() *App {
	app := &App{
		window: newWindow(),
		header: newHeader(),
	}

	return app
}

func (app *App) AddService(svc cchat.Service) {
	app.window.Services.AddService(svc, app)
}

func (app *App) RemoveSession(string) {
	app.window.MessageView.Reset()
	app.header.SetBreadcrumb(nil)
}

func (app *App) MessageRowSelected(ses *session.Row, srv *server.Row, smsg cchat.ServerMessage) {
	// Is there an old row that we should unhighlight?
	if app.lastRowHighlighter != nil {
		app.lastRowHighlighter(false)
	}

	// Set the new row and highlight it.
	app.lastRowHighlighter = srv.Button.SetActive
	app.lastRowHighlighter(true)

	app.header.SetBreadcrumb(srv.Breadcrumb())

	// Show the messages.
	app.window.MessageView.JoinServer(ses.Session, smsg)
}

func (app *App) AuthenticateSession(container *service.Container, svc cchat.Service) {
	auth.NewDialog(svc.Name(), svc.Authenticate(), func(ses cchat.Session) {
		container.AddSession(ses)
	})
}

func (app *App) Header() gtk.IWidget {
	return app.header
}

func (app *App) Window() gtk.IWidget {
	return app.window
}
