package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/buttonoverlay"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/commander"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const IconSize = 32

// Controller extends server.RowController to add session.
type Controller interface {
	// GetService asks the controller for its service.
	GetService() cchat.Service
	// OnSessionDisconnect is called before a session is disconnected. This
	// function is used for cleanups.
	OnSessionDisconnect(*Row)
	// RowSelected is called when a server that can display messages (aka
	// implements ServerMessage) is called.
	RowSelected(*Row, *server.ServerRow, cchat.ServerMessage)
	// RestoreSession is called with the session ID to ask the controller to
	// restore it from keyring information.
	RestoreSession(*Row, string) // ID string, async
	// RemoveSession is called to ask the controller to remove the session from
	// the list of sessions.
	RemoveSession(*Row)
	// MoveSession is called to ask the controller to move the session to
	// somewhere else in the list of sessions.
	MoveSession(id, movingID string)
}

// Row represents a single session, including the button header and the
// children servers.
type Row struct {
	*server.Row
	Session   cchat.Session
	sessionID string // used for reconnection

	ctrl Controller

	cmder  *commander.Buffer
	cmdbtn *gtk.Button
}

func New(parent breadcrumb.Breadcrumber, ses cchat.Session, ctrl Controller) *Row {
	row := newRow(parent, text.Rich{}, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent breadcrumb.Breadcrumber, id, name string, ctrl Controller) *Row {
	row := newRow(parent, text.Rich{Content: name}, ctrl)
	row.sessionID = id
	row.Row.SetLoading()
	return row
}

func newRow(parent breadcrumb.Breadcrumber, name text.Rich, ctrl Controller) *Row {
	srow := server.NewRow(parent, name)
	srow.Button.SetPlaceholderIcon("user-invisible-symbolic", IconSize)
	srow.Show()

	// Bind the row to .session in CSS.
	primitives.AddClass(srow, "session")
	primitives.AddClass(srow, "server-list")

	// Make a commander button that's hidden by default in case.
	cmdbtn, _ := gtk.ButtonNewFromIconName("utilities-terminal-symbolic", gtk.ICON_SIZE_BUTTON)
	buttonoverlay.Take(srow.Button, cmdbtn, server.IconSize)
	primitives.AddClass(cmdbtn, "command-button")

	row := &Row{
		Row:    srow,
		ctrl:   ctrl,
		cmdbtn: cmdbtn,
	}

	cmdbtn.Connect("clicked", row.ShowCommander)

	return row
}

// Reset extends the server row's Reset function and resets additional states.
// It resets all states back to nil, but the session ID stays.
func (r *Row) Reset() {
	r.Row.Reset()
	r.Session = nil
	r.cmder = nil
	r.cmdbtn.Hide()
}

// RemoveSession removes itself from the session list.
func (r *Row) RemoveSession() {
	// Remove the session off the list.
	r.ctrl.RemoveSession(r)

	// Asynchrously disconnect.
	go func() {
		if err := r.Session.Disconnect(); err != nil {
			log.Error(errors.Wrap(err, "Non-fatal, failed to disconnect removed session"))
		}
	}()
}

// ReconnectSession tries to reconnect with the keyring data. This is a slow
// method but it's also a very cold path.
func (r *Row) ReconnectSession() {
	// If we haven't ever connected, then don't run. In a legitimate case, this
	// shouldn't happen.
	if r.sessionID == "" {
		return
	}

	// Set the row as loading.
	r.Row.SetLoading()
	// Try to restore the session.
	r.ctrl.RestoreSession(r, r.sessionID)
}

// DisconnectSession disconnects the current session. It does nothing if the row
// does not have a session active.
func (r *Row) DisconnectSession() {
	// No-op if no session.
	if r.Session == nil {
		return
	}

	// Call the disconnect function from the controller first.
	r.ctrl.OnSessionDisconnect(r)

	// Show visually that we're disconnected first by wiping all servers.
	r.Reset()

	// Set the offline icon to the button.
	r.Button.Image.SetPlaceholderIcon("user-invisible-symbolic", IconSize)
	// Also unselect the button.
	r.Button.SetActive(false)

	// Disable the button because we're busy disconnecting. We'll re-enable them
	// once we're done reconnecting.
	r.SetSensitive(false)

	// Try and disconnect asynchronously.
	gts.Async(func() (func(), error) {
		// Disconnect and wrap the error if any. Wrap works with a nil error.
		err := errors.Wrap(r.Session.Disconnect(), "Failed to disconnect.")
		return func() {
			// Allow access to the menu
			r.SetSensitive(true)

			// Set the menu to allow disconnection.
			r.Button.SetNormalExtraMenu([]menu.Item{
				menu.SimpleItem("Connect", r.ReconnectSession),
				menu.SimpleItem("Remove", r.RemoveSession),
			})
		}, err
	})
}

// KeyringSession returns a keyring session, or nil if the session cannot be
// saved.
func (r *Row) KeyringSession() *keyring.Session {
	return keyring.ConvertSession(r.Session, r.Button.GetText())
}

// ID returns the session ID.
func (r *Row) ID() string {
	return r.sessionID
}

// SetFailed sets the initial connect status to failed. Do note that session can
// have 2 types of loading: loading the session and loading the server list.
// This one sets the former.
func (r *Row) SetFailed(err error) {
	// SetFailed, but also add the callback to retry.
	r.Row.SetFailed(err, r.ReconnectSession)
}

// SetSession binds the session and marks the row as ready. It extends SetDone.
func (r *Row) SetSession(ses cchat.Session) {
	r.Session = ses
	r.sessionID = ses.ID()
	r.SetLabelUnsafe(ses.Name())
	r.SetIconer(ses)

	// Set the commander, if any. The function will return nil if the assertion
	// returns nil. As such, we assert with an ignored ok bool, allowing cmd to
	// be nil.
	cmd, _ := ses.(commander.SessionCommander)
	r.cmder = commander.NewBuffer(r.ctrl.GetService(), cmd)
	// Show the command button if the session actually supports the commander.
	if r.cmder != nil {
		r.cmdbtn.Show()
	}

	// Bind extra menu items before loading. These items won't be clickable
	// during loading.
	r.SetNormalExtraMenu([]menu.Item{
		menu.SimpleItem("Disconnect", r.DisconnectSession),
		menu.SimpleItem("Remove", r.RemoveSession),
	})

	// Preload now.
	r.SetServerList(ses, r)
	r.Load()
}

func (r *Row) RowSelected(server *server.ServerRow, smsg cchat.ServerMessage) {
	r.ctrl.RowSelected(r, server, smsg)
}

// BindMover binds with the ID stored in the parent container to be used in the
// method itself. The ID may or may not have to do with session.
func (r *Row) BindMover(id string) {
	primitives.BindDragSortable(r.Button, "GTK_TOGGLE_BUTTON", id, r.ctrl.MoveSession)
}

// ShowCommander shows the commander dialog, or it does nothing if session does
// not implement commander.
func (r *Row) ShowCommander() {
	if r.cmder == nil {
		return
	}
	r.cmder.ShowDialog()
}
