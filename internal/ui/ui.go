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
	"github.com/markbates/pkger"
	"github.com/pkg/errors"
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
	var container = app.window.Services.AddService(svc, app)

	// Can this session be restored? If not, exit.
	restorer, ok := container.Service.(cchat.SessionRestorer)
	if !ok {
		return
	}

	var sessions = keyring.RestoreSessions(container.Service.Name())

	for _, krs := range sessions {
		// Copy the session to avoid race conditions.
		krs := krs
		row := container.AddLoadingSession(krs.ID, krs.Name)

		go app.restoreSession(row, restorer, krs)
	}
}

// RestoreSession attempts to restore the session asynchronously.
func (app *App) RestoreSession(row *session.Row, r cchat.SessionRestorer) {
	// Get the restore data.
	ks := row.KeyringSession()
	if ks == nil {
		log.Warn(errors.New("Attempted restore in ui.go"))
		return
	}
	go app.restoreSession(row, r, *ks)
}

// synchronous op
func (app *App) restoreSession(row *session.Row, r cchat.SessionRestorer, k keyring.Session) {
	s, err := r.RestoreSession(k.Data)
	if err != nil {
		err = errors.Wrapf(err, "Failed to restore session %s (%s)", k.ID, k.Name)
		log.Error(err)

		gts.ExecAsync(func() { row.SetFailed(err) })
	} else {
		gts.ExecAsync(func() { row.SetSession(s) })
	}
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

		// Try and save all keyring sessions.
		app.SaveAllSessions(container)
	})
}

func (app *App) SaveAllSessions(container *service.Container) {
	keyring.SaveSessions(container.Service.Name(), container.KeyringSessions())
}

func (app *App) Header() gtk.IWidget {
	return app.header
}

func (app *App) Window() gtk.IWidget {
	return app.window
}
