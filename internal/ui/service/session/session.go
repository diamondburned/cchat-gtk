package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/drag"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/spinner"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/button"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/commander"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// Servicer extends server.RowController to add session.
type Servicer interface {
	// Service asks the controller for its service.
	Service() cchat.Service
	// OnSessionDisconnect is called before a session is disconnected. This
	// function is used for cleanups.
	OnSessionDisconnect(*Row)
	// SessionSelected is called when the row is clicked. The parent container
	// should change the views to show this session's *Servers.
	SessionSelected(*Row)
	// MessengerSelected is called when a server that can display messages (aka
	// implements Messenger) is called.
	MessengerSelected(*Row, *server.ServerRow)
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

// Row represents a session row entry in the session List.
type Row struct {
	*gtk.ListBoxRow
	avatar  *roundimage.Avatar
	iconBox *gtk.EventBox
	icon    *rich.Icon // nillable

	parentcrumb traverse.Breadcrumber

	Session   cchat.Session // state; nilable
	sessionID string

	Servers *Servers // accessed by View for the right view
	svcctrl Servicer

	ActionsMenu *actions.Menu // session.*

	// put commander in either a hover menu or a right click menu. maybe in the
	// headerbar as well.
	cmder *commander.Buffer

	// Unread class enum for theming.
	unreadClass primitives.ClassEnum
}

var rowCSS = primitives.PrepareClassCSS("session-row",
	button.UnreadColorDefs+`

	.session-row:last-child {
		border-radius: 0 0 14px 14px;
	}

	.session-row:selected {
		background-color: alpha(@theme_selected_bg_color, 0.5);
	}

	.session-row.unread {
		background-color: alpha(@theme_fg_color, 0.25);
	}

	.session-row.unread:selected {
		background-color: alpha(mix(
			@theme_fg_color,
			@theme_selected_bg_color,
			0.65
		),  0.85);
	}

	.session-row.mentioned {
		background-color: alpha(@mentioned, 0.25);
	}

	.session-row.mentioned:selected {
		background-color: alpha(mix(
			@theme_fg_color,
			@mentioned,
			0.65
		),  0.85);
	}

	.session-row.failed {
		background-color: alpha(red, 0.45);
	}
`)

var rowIconCSS = primitives.PrepareClassCSS("session-icon", `
	.session-icon {
		padding: 4px;
		margin:  0;
	}
`)

const IconSize = 48
const IconName = "face-plain-symbolic"

func newIcon(img rich.RoundIconContainer) *rich.Icon {
	icon := rich.NewCustomIcon(img, IconSize)
	icon.SetPlaceholderIcon(IconName, IconSize)
	icon.ShowAll()
	rowIconCSS(icon)
	return icon
}

func New(parent traverse.Breadcrumber, ses cchat.Session, ctrl Servicer) *Row {
	row := newRow(parent, text.Rich{}, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent traverse.Breadcrumber, id, name string, ctrl Servicer) *Row {
	row := newRow(parent, text.Rich{Content: name}, ctrl)
	row.sessionID = id
	row.SetLoading()
	return row
}

func newRow(parent traverse.Breadcrumber, name text.Rich, ctrl Servicer) *Row {
	row := &Row{
		svcctrl:     ctrl,
		parentcrumb: parent,
	}

	row.avatar = roundimage.NewAvatar(IconSize)
	row.avatar.SetText(name.Content)
	row.avatar.Show()

	row.iconBox, _ = gtk.EventBoxNew()
	row.iconBox.Show()

	row.ListBoxRow, _ = gtk.ListBoxRowNew()
	row.ListBoxRow.Show()
	rowCSS(row.ListBoxRow)

	// TODO: commander button

	row.Servers = NewServers(row, row)
	row.Servers.Show()

	// Bind session.* actions into row.
	row.ActionsMenu = actions.NewMenu("session")
	row.ActionsMenu.InsertActionGroup(row)

	// Bind right clicks and show a popover menu on such event.
	row.iconBox.Connect("button-press-event", func(_ gtk.IWidget, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			row.ActionsMenu.Popup(row)
		}
	})

	// Bind drag-and-drop events.
	drag.BindDraggable(row, "face-smile", ctrl.MoveSession)

	// Bind the unread state.
	row.Servers.Children.SetUnreadHandler(func(unread, mentioned bool) {
		switch {
		// Prioritize mentions over unreads.
		case mentioned:
			row.unreadClass.SetClass(row, "mentioned")
		case unread:
			row.unreadClass.SetClass(row, "unread")
		default:
			row.unreadClass.SetClass(row, "read")
		}
	})

	// Reset to bring states set in that method to a newly constructed widget.
	row.Reset()

	return row
}

func NewAddButton() *gtk.ListBoxRow {
	img, _ := gtk.ImageNew()
	img.Show()
	primitives.SetImageIcon(img, "list-add-symbolic", IconSize/2)

	row, _ := gtk.ListBoxRowNew()
	row.SetSizeRequest(IconSize, IconSize)
	row.SetSelectable(false) // activatable though
	row.Add(img)
	row.Show()
	rowCSS(row)

	return row
}

// Reset extends the server row's Reset function and resets additional states.
// It resets all states back to nil, but the session ID stays.
func (r *Row) Reset() {
	r.Servers.Reset()     // wipe servers
	r.ActionsMenu.Reset() // wipe menu items
	r.ActionsMenu.AddAction("Remove", r.RemoveSession)

	if r.icon == nil {
		r.icon = newIcon(r.avatar)
		r.iconBox.Add(r.icon)
	}

	// Set a lame placeholder icon.
	r.icon.SetPlaceholderIcon("folder-remote-symbolic", IconSize)

	r.Session = nil
	r.cmder = nil
}

func (r *Row) ParentBreadcrumb() traverse.Breadcrumber {
	return r.parentcrumb
}

func (r *Row) Breadcrumb() string {
	if r.Session == nil {
		return ""
	}
	return r.Session.Name().Content
}

// Activate executes whatever needs to be done. If the row has failed, then this
// method will reconnect. If the row is already loaded, then SessionSelected
// will be called.
func (r *Row) Activate() {
	// If session is nil, then we've probably failed to load it. The row is
	// deactivated while loading, so this wouldn't have happened.
	if r.Session == nil {
		r.ReconnectSession()
	} else {
		// Load all servers in this root node, then call the parent controller's
		// method.
		r.Servers.Children.LoadAll()
	}

	// Display the empty server list first, then try and reconnect.
	r.svcctrl.SessionSelected(r)
}

// SetLoading sets the session button to have a spinner circle. DO NOT CONFUSE
// THIS WITH THE SERVERS LOADING.
func (r *Row) SetLoading() {
	// Reset the state.
	r.Session = nil

	// Reset the icon.
	primitives.RemoveChildren(r.iconBox)
	r.icon = nil

	// Remove everything from the row, including the icon.
	primitives.RemoveChildren(r)

	// Remove the failed class.
	primitives.RemoveClass(r, "failed")

	// Add a loading circle.
	spin := spinner.New()
	spin.SetSizeRequest(IconSize, IconSize)
	spin.Start()
	spin.Show()
	rowIconCSS(spin)

	r.Add(spin)
	r.SetSensitive(false) // no activate
}

// SetFailed sets the initial connect status to failed. Do note that session can
// have 2 types of loading: loading the session and loading the server list.
// This one sets the former.
func (r *Row) SetFailed(err error) {
	// Make sure that Session is still nil.
	r.Session = nil
	// Re-enable the row.
	r.SetSensitive(true)
	// Remove everything off the row.
	primitives.RemoveChildren(r)
	// Mark the row as failed.
	primitives.AddClass(r, "failed")

	if r.icon == nil {
		r.icon = newIcon(r.avatar)
		r.iconBox.Add(r.icon)
	}

	// Add the icon.
	r.Add(r.iconBox)
	// Set the button to a retry icon.
	r.icon.SetPlaceholderIcon("view-refresh-symbolic", IconSize)
}

func (r *Row) RestoreSession(res cchat.SessionRestorer, k keyring.Session) {
	go func() {
		s, err := res.RestoreSession(k.Data)
		if err != nil {
			err = errors.Wrapf(err, "failed to restore session %s (%s)", k.ID, k.Name)
			log.Error(err)

			gts.ExecAsync(func() { r.SetFailed(err) })
		} else {
			gts.ExecAsync(func() { r.SetSession(s) })
		}
	}()
}

// SetSession binds the session and marks the row as ready. It extends SetDone.
func (r *Row) SetSession(ses cchat.Session) {
	// Set the states.
	r.Session = ses
	r.sessionID = ses.ID()
	r.SetTooltipMarkup(markup.Render(ses.Name()))
	r.avatar.SetText(ses.Name().Content)

	if r.icon == nil {
		r.icon = newIcon(r.avatar)
		r.iconBox.Add(r.icon)
	}

	r.icon.SetPlaceholderIcon(IconName, IconSize)

	// If the session has an icon, then use it.
	if iconer := ses.AsIconer(); iconer != nil {
		r.icon.AsyncSetIconer(iconer, "failed to set session icon")
	}

	// Update to indicate that we're done.
	primitives.RemoveChildren(r)
	r.SetSensitive(true)
	r.Add(r.iconBox)

	// Bind extra menu items before loading. These items won't be clickable
	// during loading.
	r.ActionsMenu.Reset()
	r.ActionsMenu.AddAction("Disconnect", r.DisconnectSession)
	r.ActionsMenu.AddAction("Remove", r.RemoveSession)

	// Set the commander, if any. The function will return nil if the assertion
	// returns nil. As such, we assert with an ignored ok bool, allowing cmd to
	// be nil.
	if cmder := ses.AsCommander(); cmder != nil {
		r.cmder = commander.NewBuffer(ses.Name().String(), cmder)
		// Show the command button if the session actually supports the
		// commander.
		r.ActionsMenu.AddAction("Command Prompt", r.ShowCommander)
	}

	// Load all top-level servers now.
	r.Servers.SetList(ses)
}

func (r *Row) MessengerSelected(sr *server.ServerRow) {
	r.svcctrl.MessengerSelected(r, sr)
}

// RemoveSession removes itself from the session list.
func (r *Row) RemoveSession() {
	// Remove the session off the list.
	r.svcctrl.RemoveSession(r)

	var session = r.Session
	if session == nil {
		return
	}

	// Asynchrously disconnect.
	go func() {
		if err := session.Disconnect(); err != nil {
			log.Error(errors.Wrap(err, "non-fatal; failed to disconnect removed session"))
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
	r.SetLoading()
	// Try to restore the session.
	r.svcctrl.RestoreSession(r, r.sessionID)
}

// DisconnectSession disconnects the current session. It does nothing if the row
// does not have a session active.
func (r *Row) DisconnectSession() {
	// No-op if no session.
	if r.Session == nil {
		return
	}

	// Call the disconnect function from the controller first.
	r.svcctrl.OnSessionDisconnect(r)

	// Copy the session to avoid data race and allow us to reset.
	session := r.Session

	// Show visually that we're disconnected first by wiping all servers.
	r.Reset()

	// Disable the button because we're busy disconnecting. We'll re-enable them
	// once we're done reconnecting.
	r.SetSensitive(false)

	// Try and disconnect asynchronously.
	gts.Async(func() (func(), error) {
		// Disconnect and wrap the error if any. Wrap works with a nil error.
		err := errors.Wrap(session.Disconnect(), "failed to disconnect.")
		return func() {
			// Re-enable access to the menu.
			r.SetSensitive(true)

			// Set the menu to allow disconnection.
			r.ActionsMenu.AddAction("Connect", r.ReconnectSession)
			r.ActionsMenu.AddAction("Remove", r.RemoveSession)
		}, err
	})
}

// ID returns the session ID.
func (r *Row) ID() string {
	return r.sessionID
}

// ShowCommander shows the commander dialog, or it does nothing if session does
// not implement commander.
func (r *Row) ShowCommander() {
	if r.cmder == nil {
		return
	}
	r.cmder.ShowDialog()
}
