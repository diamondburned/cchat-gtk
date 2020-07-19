package ui

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config/preferences"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/auth"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func init() {
	// Load the local CSS.
	gts.LoadCSS("main", `
		/* Make CSS more consistent across themes */
		headerbar { padding-left: 0 }

		/* .appmenu { margin: 0 20px } */
		
		popover > *:not(stack):not(button) { margin: 6px }
		
		/* Hack to fix the input bar being high in Adwaita */
		.input-field * { min-height: 0 }
	`)
}

// constraints for the left panel
const (
	leftMinWidth     = 200
	leftCurrentWidth = 275
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
	_ gts.Window         = (*App)(nil)
	_ service.Controller = (*App)(nil)
)

func NewApplication() *App {
	app := &App{}
	app.window = newWindow(app)
	app.header = newHeader()

	// Resize the app icon with the left-most sidebar.
	services := app.window.Services.Services
	services.Connect("size-allocate", func() {
		app.header.left.appmenu.SetSizeRequest(services.GetAllocatedWidth(), -1)
	})

	// Resize the left-side header w/ the left-side pane.
	app.window.Services.ServerView.Connect("size-allocate", func() {
		// Get the current width of the left sidebar.
		width := app.window.GetPosition()
		// Set the left-side header's size.
		app.header.left.SetSizeRequest(width, -1)
	})

	// Bind the preferences action for our GAction button in the header popover.
	// The action name for this is "app.preferences".
	gts.AddAppAction("preferences", preferences.SpawnPreferenceDialog)

	return app
}

func (app *App) AddService(svc cchat.Service) {
	app.window.Services.AddService(svc)
}

// OnSessionRemove resets things before the session is removed.
func (app *App) OnSessionRemove(s *service.Service, r *session.Row) {
	// Reset the message view if it's what we're showing.
	if app.window.MessageView.SessionID() == r.ID() {
		app.window.MessageView.Reset()
		app.header.SetBreadcrumber(nil)
	}
}

func (app *App) OnSessionDisconnect(s *service.Service, r *session.Row) {
	// We're basically doing the same thing as removing a session. Check
	// OnSessionRemove above.
	app.OnSessionRemove(s, r)
}

func (app *App) SessionSelected(svc *service.Service, ses *session.Row) {
	// Is there an old row that we should deactivate?
	if app.lastSelector != nil {
		app.lastSelector(false)
		app.lastSelector = nil
	}

	// TODO: restore last message box
	app.window.MessageView.Reset()
	app.header.SetBreadcrumber(ses)
	app.header.SetSessionMenu(ses)
}

func (app *App) RowSelected(ses *session.Row, srv *server.ServerRow, smsg cchat.ServerMessage) {
	// Is there an old row that we should deactivate?
	if app.lastSelector != nil {
		app.lastSelector(false)
	}

	// Set the new row.
	app.lastSelector = srv.SetSelected
	app.lastSelector(true)

	app.header.SetBreadcrumber(srv)

	// Disable the server list because we don't want the user to switch around
	// while we're loading.
	app.window.Services.SetSensitive(false)

	// Assert that server is also a list, then join the server.
	app.window.MessageView.JoinServer(ses.Session, smsg.(messages.ServerMessage), func() {
		// Re-enable the server list.
		app.window.Services.SetSensitive(true)
	})
}

func (app *App) AuthenticateSession(list *service.List, ssvc *service.Service) {
	var svc = ssvc.Service()
	auth.NewDialog(svc.Name(), svc.Authenticate(), func(ses cchat.Session) {
		ssvc.AddSession(ses)
	})
}

// Close is called when the application finishes gracefully.
func (app *App) Close() {
	// Disconnect everything. This blocks the main thread, so by the time we're
	// done, the application would exit immediately. There's no need to update
	// the GUI.
	for _, s := range app.window.AllServices() {
		var service = s.Service().Name()

		for _, session := range s.BodyList.Sessions() {
			if session.Session == nil {
				continue
			}

			log.Printlnf("Disconnecting %s session %s", service, session.ID())

			if err := session.Session.Disconnect(); err != nil {
				log.Error(errors.Wrap(err, "Failed to disconnect "+session.ID()))
			}
		}
	}
}

func (app *App) Header() gtk.IWidget {
	return app.header
}

func (app *App) Window() gtk.IWidget {
	return app.window
}

func (app *App) Icon() *gdk.Pixbuf {
	return icons.Logo256(0)
}

func (app *App) Menu() *glib.MenuModel {
	return &app.header.menu.MenuModel
}
