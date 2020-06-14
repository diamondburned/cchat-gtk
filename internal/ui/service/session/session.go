package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const IconSize = 32

// Controller extends server.RowController to add session.
type Controller interface {
	// OnSessionDisconnect is called before a session is disconnected. This
	// function is used for cleanups.
	OnSessionDisconnect(*Row)
	// MessageRowSelected is called when a server that can display messages (aka
	// implements ServerMessage) is called.
	MessageRowSelected(*Row, *server.Row, cchat.ServerMessage)
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
	*gtk.Box
	Button  *rich.ToggleButtonImage
	Session cchat.Session
	Servers *server.Children

	ctrl       Controller
	parent     breadcrumb.Breadcrumber
	menuconstr func(*gtk.Menu)
	sessionID  string // used for reconnection

	// nil after calling SetSession()
	// krs keyring.Session
}

func New(parent breadcrumb.Breadcrumber, ses cchat.Session, ctrl Controller) *Row {
	row := newRow(parent, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent breadcrumb.Breadcrumber, id, name string, ctrl Controller) *Row {
	row := newRow(parent, ctrl)
	row.sessionID = id
	row.Button.SetLabelUnsafe(text.Rich{Content: name})
	row.setLoading()

	return row
}

var dragEntries = []gtk.TargetEntry{
	primitives.NewTargetEntry("GTK_TOGGLE_BUTTON"),
}
var dragAtom = gdk.GdkAtomIntern("GTK_TOGGLE_BUTTON", true)

func newRow(parent breadcrumb.Breadcrumber, ctrl Controller) *Row {
	row := &Row{
		ctrl:   ctrl,
		parent: parent,
	}
	row.Servers = server.NewChildren(parent, row)
	row.Servers.SetLoading()

	row.Button = rich.NewToggleButtonImage(text.Rich{})
	row.Button.Box.SetHAlign(gtk.ALIGN_START)
	row.Button.Image.AddProcessors(imgutil.Round(true))
	// Set the loading icon.
	row.Button.SetRelief(gtk.RELIEF_NONE)
	row.Button.Show()

	// On click, toggle reveal.
	row.Button.Connect("clicked", func() {
		revealed := !row.Servers.GetRevealChild()
		row.Servers.SetRevealChild(revealed)
		row.Button.SetActive(revealed)
	})

	row.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	row.Box.SetMarginStart(server.ChildrenMargin)
	row.Box.PackStart(row.Button, false, false, 0)
	row.Box.Show()

	// Bind the box to .session in CSS.
	primitives.AddClass(row.Box, "session")
	// Bind the button to create a new menu.
	primitives.BindDynamicMenu(row.Button, func(menu *gtk.Menu) {
		row.menuconstr(menu)
	})

	// noop, empty menu
	row.menuconstr = func(menu *gtk.Menu) {}

	return row
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
	// If we haven't ever connected:
	if r.sessionID == "" {
		return
	}

	r.setLoading()
	r.ctrl.RestoreSession(r, r.sessionID)
}

// DisconnectSession disconnects the current session.
func (r *Row) DisconnectSession() {
	// Call the disconnect function from the controller first.
	r.ctrl.OnSessionDisconnect(r)

	// Show visually that we're disconnected first by wiping all servers.
	r.Box.Remove(r.Servers)
	r.Servers.Reset()

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
			// allow access to the menu
			r.SetSensitive(true)

			// set the menu to allow disconnection.
			r.menuconstr = func(menu *gtk.Menu) {
				primitives.AppendMenuItems(menu, []gtk.IMenuItem{
					primitives.MenuItem("Connect", r.ReconnectSession),
					primitives.MenuItem("Remove", r.RemoveSession),
				})
			}
		}, err
	})
}

func (r *Row) setLoading() {
	// set the loading icon
	r.Button.Image.SetPlaceholderIcon("content-loading-symbolic", IconSize)
	// set the loading icon in the servers list
	r.Servers.SetLoading()
	// restore the old label's color
	r.Button.SetLabelUnsafe(r.Button.GetLabel())
	// clear the tooltip
	r.SetTooltipText("")
	// blur - set the color darker
	r.SetSensitive(false)
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

func (r *Row) SetSession(ses cchat.Session) {
	r.Session = ses
	r.sessionID = ses.ID()

	r.Servers.SetServerList(ses)
	r.Box.PackStart(r.Servers, false, false, 0)

	r.Button.SetLabelUnsafe(ses.Name())
	r.Button.Image.SetPlaceholderIcon("user-available-symbolic", IconSize)

	r.SetSensitive(true)
	r.SetTooltipText("") // reset

	// Try and set the session's icon.
	if iconer, ok := ses.(cchat.Icon); ok {
		r.Button.Image.AsyncSetIcon(iconer.Icon, "Error fetching session icon URL")
	}

	// Set the menu with the disconnect button.
	r.menuconstr = func(menu *gtk.Menu) {
		primitives.AppendMenuItems(menu, []gtk.IMenuItem{
			primitives.MenuItem("Disconnect", r.DisconnectSession),
			primitives.MenuItem("Remove", r.RemoveSession),
		})
	}
}

func (r *Row) SetFailed(err error) {
	// Allow the retry button to be pressed.
	r.menuconstr = func(menu *gtk.Menu) {
		primitives.AppendMenuItems(menu, []gtk.IMenuItem{
			primitives.MenuItem("Retry", r.ReconnectSession),
			primitives.MenuItem("Remove", r.RemoveSession),
		})
	}

	r.SetSensitive(true)
	r.SetTooltipText(err.Error())
	// Intentional side-effect of not changing the actual label state.
	r.Button.Label.SetMarkup(rich.MakeRed(r.Button.GetLabel()))
	// Set the icon to a failed one.
	r.Button.Image.SetPlaceholderIcon("computer-fail-symbolic", IconSize)
}

func (r *Row) MessageRowSelected(server *server.Row, smsg cchat.ServerMessage) {
	r.ctrl.MessageRowSelected(r, server, smsg)
}

func (r *Row) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(r.parent, r.Button.GetLabel().Content)
}

// BindMover binds with the ID stored in the parent container to be used in the
// method itself. The ID may or may not have to do with session.
func (r *Row) BindMover(id string) {
	primitives.BindDragSortable(r.Button, "GTK_TOGGLE_BUTTON", id, r.ctrl.MoveSession)
}
