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

// constraints for the left panel
const (
	leftMinWidth     = 200
	leftCurrentWidth = 250
	leftMaxWidth     = 400
)

func clamp(n, min, max int) int {
	switch {
	case n > max:
		return max
	case n < min:
		return min
	default:
		return n
	}
}

type App struct {
	window *window
	header *header

	// used to keep track of what row to disconnect before switching
	lastSelector func(bool)
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

	// Resize the left-side header w/ the left-side pane.
	app.window.Services.Connect("size-allocate", func(wv gtk.IWidget) {
		// Get the current width of the left sidebar.
		var width = app.window.GetPosition()
		// Set the left-side header's size.
		app.header.left.SetSizeRequest(width, -1)
	})

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

func (app *App) OnSessionDisconnect(id string) {
	// We're basically doing the same thing as removing a session. Check
	// OnSessionRemove above.
	app.OnSessionRemove(id)
}

func (app *App) RowSelected(ses *session.Row, srv *server.ServerRow, smsg cchat.ServerMessage) {
	// Is there an old row that we should deactivate?
	if app.lastSelector != nil {
		app.lastSelector(false)
	}

	// Set the new row.
	app.lastSelector = srv.SetSelected
	app.lastSelector(true)

	app.header.SetBreadcrumb(srv.Breadcrumb())

	// Disable the server list because we don't want the user to switch around
	// while we're loading.
	app.window.Services.SetSensitive(false)

	// Assert that server is also a list, then join the server.
	app.window.MessageView.JoinServer(ses.Session, smsg.(messages.ServerMessage), func() {
		// Re-enable the server list.
		app.window.Services.SetSensitive(true)
	})
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
