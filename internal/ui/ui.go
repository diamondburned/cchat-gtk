package ui

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/auth"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
)

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
	var container = app.window.Services.AddService(svc, app)

	// Attempt to restore sessions asynchronously.
	keyring.RestoreSessions(svc, container.AddSession)
}

func (app *App) MessageRowSelected(ses *session.Row, srv *server.Row, smsg cchat.ServerMessage) {
	// Is there an old row that we should unhighlight?
	if app.lastRowHighlighter != nil {
		app.lastRowHighlighter(false)
	}

	// Set the new row and highlight it.
	app.lastRowHighlighter = srv.Button.SetActive
	app.lastRowHighlighter(true)

	log.Println("Breadcrumb:")

	// Show the messages.
	app.window.MessageView.JoinServer(ses.Session, smsg)
}

func (app *App) AuthenticateSession(container *service.Container, svc cchat.Service) {
	auth.NewDialog(svc.Name(), svc.Authenticate(), func(ses cchat.Session) {
		container.AddSession(ses)

		// Save all sessions.
		for _, err := range keyring.SaveSessions(svc, container.Sessions()) {
			log.Error(err)
		}
	})
}

func (app *App) Header() gtk.IWidget {
	return app.header
}

func (app *App) Window() gtk.IWidget {
	return app.window
}
