package ui

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config/preferences"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/auth"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/handy"
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

		/* Hide all scroll undershoots */
		undershoot { background-size: 0 }
	`)
}

// constraints for the left panel
const leftCurrentWidth = 300

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
	handy.Leaflet
	HeaderGroup *handy.HeaderGroup

	Services    *service.View
	MessageView *messages.View

	// used to keep track of what row to disconnect before switching
	lastSelector func(bool)
}

var (
	_ gts.MainApplication = (*App)(nil)
	_ service.Controller  = (*App)(nil)
	_ messages.Controller = (*App)(nil)
)

func NewApplication() *App {
	app := &App{}

	app.Services = service.NewView(app)
	app.Services.SetSizeRequest(leftCurrentWidth, -1)
	app.Services.SetHExpand(false)
	app.Services.Show()

	app.MessageView = messages.NewView(app)
	app.MessageView.SetHExpand(true)
	app.MessageView.Show()

	app.HeaderGroup = handy.HeaderGroupNew()
	app.HeaderGroup.AddHeaderBar(&app.Services.Header.HeaderBar)
	app.HeaderGroup.AddHeaderBar(&app.MessageView.Header.HeaderBar)

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	separator.Show()

	app.Leaflet = *handy.LeafletNew()
	app.Leaflet.SetChildTransitionDuration(75)
	app.Leaflet.SetTransitionType(handy.LeafletTransitionTypeSlide)
	app.Leaflet.SetCanSwipeBack(true)

	app.Leaflet.Add(app.Services)
	app.Leaflet.Add(separator)
	app.Leaflet.Add(app.MessageView)

	app.Leaflet.ChildSetProperty(separator, "navigatable", false)
	app.Leaflet.Show()

	// Bind the preferences action for our GAction button in the header popover.
	// The action name for this is "app.preferences".
	gts.AddAppAction("preferences", preferences.SpawnPreferenceDialog)

	primitives.LeafletOnFold(&app.Leaflet, app.MessageView.SetFolded)

	return app
}

// Services methods.

func (app *App) AddService(svc cchat.Service) {
	app.Services.AddService(svc)
}

// OnSessionRemove resets things before the session is removed.
func (app *App) OnSessionRemove(s *service.Service, r *session.Row) {
	// Reset the message view if it's what we're showing.
	if app.MessageView.SessionID() == r.ID() {
		app.MessageView.Reset()
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

	// TODO
	// reset view when setservers top level called

	// TODO: restore last message box
	app.MessageView.Reset()
}

func (app *App) ClearMessenger(ses *session.Row) {
	if app.MessageView.SessionID() == ses.Session.ID() {
		return
	}
}

func (app *App) MessengerSelected(ses *session.Row, srv *server.ServerRow) {
	// Change to the message view.
	app.Leaflet.SetVisibleChild(app.MessageView)

	// Assert that the new server is not the same one.
	if app.MessageView.SessionID() == ses.Session.ID() &&
		app.MessageView.ServerID() == srv.Server.ID() {

		return
	}

	// Is there an old row that we should deactivate?
	if app.lastSelector != nil {
		app.lastSelector(false)
	}

	// Set the new row.
	app.lastSelector = srv.SetSelected
	app.lastSelector(true)

	app.MessageView.JoinServer(ses.Session, srv.Server, srv)
}

// MessageView methods.

func (app *App) GoBack() {
	app.Leaflet.Navigate(handy.NavigationDirectionBack)
}

func (app *App) OnMessageBusy() {
	// Disable the server list because we don't want the user to switch around
	// while we're loading.
	app.Services.SetSensitive(false)
}

func (app *App) OnMessageDone() {
	// Re-enable the server list.
	app.Services.SetSensitive(true)
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
	for _, s := range app.Services.Services.Services {
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

func (app *App) Icon() *gdk.Pixbuf {
	return icons.Logo256Pixbuf()
}

func (app *App) Menu() *glib.MenuModel {
	return app.Services.Header.MenuModel
}
