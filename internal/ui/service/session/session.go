package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/spinner"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/commander"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const IconSize = 48
const IconName = "face-plain-symbolic"

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

// Row represents a session row entry in the session List.
type Row struct {
	*gtk.ListBoxRow
	icon *rich.EventIcon // nilable

	parentcrumb breadcrumb.Breadcrumber

	Session   cchat.Session // state; nilable
	sessionID string

	Servers *Servers // accessed by View for the right view
	svcctrl Servicer

	ActionsMenu *actions.Menu // session.*

	// TODO: enum class? having the button be red on fail would be good

	// put commander in either a hover menu or a right click menu. maybe in the
	// headerbar as well.
	// TODO headerbar how? custom interface to get menu items and callbacks in
	// controller?
	cmder *commander.Buffer
}

var rowCSS = primitives.PrepareClassCSS("session-row", `
	.session-row:last-child {
		border-radius: 0 0 14px 14px;
	}
	.session-row:selected {
		background-color: alpha(@theme_selected_bg_color, 0.5);
	}
`)

var rowIconCSS = primitives.PrepareClassCSS("session-icon", `
	.session-icon {
		padding: 4px;
		margin:  0;
	}
	.session-icon.failed {
		background-color: alpha(red, 0.45);
	}
`)

func New(parent breadcrumb.Breadcrumber, ses cchat.Session, ctrl Servicer) *Row {
	row := newRow(parent, text.Rich{}, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent breadcrumb.Breadcrumber, id, name string, ctrl Servicer) *Row {
	row := newRow(parent, text.Rich{Content: name}, ctrl)
	row.sessionID = id
	row.SetLoading()
	return row
}

func newRow(parent breadcrumb.Breadcrumber, name text.Rich, ctrl Servicer) *Row {
	row := &Row{
		svcctrl:     ctrl,
		parentcrumb: parent,
	}

	row.icon = rich.NewEventIcon(IconSize)
	row.icon.Icon.SetPlaceholderIcon(IconName, IconSize)
	row.icon.Show()
	rowIconCSS(row.icon.Icon)

	row.ListBoxRow, _ = gtk.ListBoxRowNew()
	rowCSS(row.ListBoxRow)

	// TODO: commander button

	row.Servers = NewServers(row, row)
	row.Servers.Show()

	// Bind session.* actions into row.
	row.ActionsMenu = actions.NewMenu("session")
	row.ActionsMenu.InsertActionGroup(row)

	// Bind right clicks and show a popover menu on such event.
	row.icon.Connect("button-press-event", func(_ gtk.IWidget, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			row.ActionsMenu.Popover(row).Popup()
		}
	})

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

	// Set a lame placeholder icon.
	r.icon.Icon.SetPlaceholderIcon("folder-remote-symbolic", IconSize)

	r.Session = nil
	r.cmder = nil
}

func (r *Row) SessionID() string {
	return r.sessionID
}

func (r *Row) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(r.parentcrumb, r.Session.Name().Content)
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
		r.svcctrl.SessionSelected(r)
	}
}

// SetLoading sets the session button to have a spinner circle. DO NOT CONFUSE
// THIS WITH THE SERVERS LOADING.
func (r *Row) SetLoading() {
	// Reset the state.
	r.Session = nil

	// Reset the icon.
	r.icon.Icon.Reset()

	// Remove everything from the row, including the icon.
	primitives.RemoveChildren(r)

	// Remove the failed class.
	primitives.RemoveClass(r.icon.Icon, "failed")

	// Add a loading circle.
	spin := spinner.New()
	spin.SetSizeRequest(IconSize, IconSize)
	spin.Start()
	spin.Show()

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
	// Add the icon.
	r.Add(r.icon)
	// Set the button to a retry icon.
	r.icon.Icon.SetPlaceholderIcon("view-refresh-symbolic", IconSize)
	// Mark the icon as failed.
	primitives.AddClass(r.icon.Icon, "failed")

	// SetFailed, but also add the callback to retry.
	// r.Row.SetFailed(err, r.ReconnectSession)
}

func (r *Row) RestoreSession(res cchat.SessionRestorer, k keyring.Session) {
	go func() {
		s, err := res.RestoreSession(k.Data)
		if err != nil {
			err = errors.Wrapf(err, "Failed to restore session %s (%s)", k.ID, k.Name)
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
	r.icon.Icon.SetPlaceholderIcon(IconName, IconSize)

	// If the session has an icon, then use it.
	if iconer, ok := ses.(cchat.Icon); ok {
		r.icon.Icon.AsyncSetIconer(iconer, "Failed to set session icon")
	}

	// Update to indicate that we're done.
	primitives.RemoveChildren(r)
	r.SetSensitive(true)
	r.Add(r.icon)

	// Bind extra menu items before loading. These items won't be clickable
	// during loading.
	r.ActionsMenu.Reset()
	r.ActionsMenu.AddAction("Disconnect", r.DisconnectSession)
	r.ActionsMenu.AddAction("Remove", r.RemoveSession)

	// Set the commander, if any. The function will return nil if the assertion
	// returns nil. As such, we assert with an ignored ok bool, allowing cmd to
	// be nil.
	cmd, _ := ses.(commander.SessionCommander)
	r.cmder = commander.NewBuffer(r.svcctrl.Service(), cmd)

	// Show the command button if the session actually supports the commander.
	if r.cmder != nil {
		r.ActionsMenu.AddAction("Command Prompt", r.ShowCommander)
	}

	// Load all top-level servers now.
	r.Servers.SetList(ses)
}

// BindMover binds with the ID stored in the parent container to be used in the
// method itself. The ID may or may not have to do with session.
func (r *Row) BindMover(id string) {
	// TODO: rows can be highlighted.
	// primitives.BindDragSortable(r.Button, "GTK_TOGGLE_BUTTON", id, r.ctrl.MoveSession)
}

func (r *Row) RowSelected(sr *server.ServerRow, smsg cchat.ServerMessage) {
	r.svcctrl.RowSelected(r, sr, smsg)
}

// RemoveSession removes itself from the session list.
func (r *Row) RemoveSession() {
	// Remove the session off the list.
	r.svcctrl.RemoveSession(r)

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
		err := errors.Wrap(session.Disconnect(), "Failed to disconnect.")
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

// deprecate server.Row inheritance since the structure is entirely different

/*
// Row represents a single session, including the button header and the
// children servers.
type Row struct {
	*server.Row
	Session   cchat.Session
	sessionID string // used for reconnection

	ctrl Servicer

	cmder  *commander.Buffer
	cmdbtn *gtk.Button
}

func New(parent breadcrumb.Breadcrumber, ses cchat.Session, ctrl Servicer) *Row {
	row := newRow(parent, text.Rich{}, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent breadcrumb.Breadcrumber, id, name string, ctrl Servicer) *Row {
	row := newRow(parent, text.Rich{Content: name}, ctrl)
	row.sessionID = id
	row.Row.SetLoading()
	return row
}

func newRow(parent breadcrumb.Breadcrumber, name text.Rich, ctrl Servicer) *Row {
	srow := server.NewRow(parent, name)
	srow.Button.SetPlaceholderIcon(IconName, IconSize)
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
	r.Button.Image.SetPlaceholderIcon(IconName, IconSize)
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

// ShowCommander shows the commander dialog, or it does nothing if session does
// not implement commander.
func (r *Row) ShowCommander() {
	if r.cmder == nil {
		return
	}
	r.cmder.ShowDialog()
}
*/
