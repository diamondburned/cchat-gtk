package ui

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages"
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

	// used to keep track of what row to disconnect before switching
	lastDeactivator func()
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

// OnSessionRemove resets things before the session is removed.
func (app *App) OnSessionRemove(id string) {
	// Reset the message view if it's what we're showing.
	if app.window.MessageView.SessionID() == id {
		app.window.MessageView.Reset()
		app.header.SetBreadcrumb(nil)
	}
}

func (app *App) MessageRowSelected(ses *session.Row, srv *server.Row, smsg cchat.ServerMessage) {
	// Is there an old row that we should deactivate?
	if app.lastDeactivator != nil {
		app.lastDeactivator()
	}
	// Set the new row.
	app.lastDeactivator = srv.Deactivate

	app.header.SetBreadcrumb(srv.Breadcrumb())

	// Assert that server is also a list, then join the server.
	app.window.MessageView.JoinServer(ses.Session, smsg.(messages.ServerMessage))
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
