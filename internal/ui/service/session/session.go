package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const IconSize = 32

// Controller extends server.RowController to add session.
type Controller interface {
	MessageRowSelected(*Row, *server.Row, cchat.ServerMessage)
	RestoreSession(*Row, keyring.Session) // async
	RemoveSession(*Row)
	MoveSession(id, movingID string)
}

// Row represents a single session, including the button header and the
// children servers.
type Row struct {
	*gtk.Box
	Button  *rich.ToggleButtonImage
	Session cchat.Session
	Servers *server.Children

	menu  *gtk.Menu
	retry *gtk.MenuItem

	ctrl   Controller
	parent breadcrumb.Breadcrumber

	// nil after calling SetSession()
	krs keyring.Session
}

func New(parent breadcrumb.Breadcrumber, ses cchat.Session, ctrl Controller) *Row {
	row := new(parent, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent breadcrumb.Breadcrumber, name string, ctrl Controller) *Row {
	row := new(parent, ctrl)
	row.Button.SetLabelUnsafe(text.Rich{Content: name})
	row.setLoading()

	return row
}

var dragEntries = []gtk.TargetEntry{
	primitives.NewTargetEntry("GTK_TOGGLE_BUTTON"),
}
var dragAtom = gdk.GdkAtomIntern("GTK_TOGGLE_BUTTON", true)

func new(parent breadcrumb.Breadcrumber, ctrl Controller) *Row {
	row := &Row{
		ctrl:   ctrl,
		parent: parent,
	}
	row.Servers = server.NewChildren(parent, row)

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

	primitives.AddClass(row.Box, "session")

	row.menu, _ = gtk.MenuNew()
	primitives.BindMenu(row.menu, row.Button)

	row.retry = primitives.HiddenMenuItem("Retry", func() {
		// Show the loading stuff.
		row.setLoading()
		// Reuse the failed keyring session provided. As this variable is reset
		// after a success, it relies of the button not triggering.
		ctrl.RestoreSession(row, row.krs)
	})
	row.retry.SetSensitive(false)

	primitives.AppendMenuItems(row.menu, []*gtk.MenuItem{
		row.retry,
		primitives.MenuItem("Remove", func() {
			ctrl.RemoveSession(row)
		}),
	})

	return row
}

func (r *Row) setLoading() {
	// set the loading icon
	r.Button.Image.SetPlaceholderIcon("content-loading-symbolic", IconSize)
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
	return keyring.GetSession(r.Session, r.Button.GetText())
}

func (r *Row) SetSession(ses cchat.Session) {
	// Disable the retry button.
	r.retry.SetSensitive(false)
	r.retry.Hide()

	r.Session = ses
	r.Servers.SetServerList(ses)
	r.Button.SetLabelUnsafe(ses.Name())
	r.Button.Image.SetPlaceholderIcon("user-available-symbolic", IconSize)
	r.Box.PackStart(r.Servers, false, false, 0)
	r.SetSensitive(true)
	r.SetTooltipText("") // reset

	// Try and set the session's icon.
	if iconer, ok := ses.(cchat.Icon); ok {
		r.Button.Image.AsyncSetIcon(iconer.Icon, "Error fetching session icon URL")
	}

	// Wipe the keyring session off.
	r.krs = keyring.Session{}
}

func (r *Row) SetFailed(krs keyring.Session, err error) {
	// Set the failed keyring session.
	r.krs = krs

	// Allow the retry button to be pressed.
	r.retry.SetSensitive(true)
	r.retry.Show()

	r.SetSensitive(true)
	r.SetTooltipText(err.Error())
	// Intentional side-effect of not changing the actual label state.
	r.Button.Label.SetMarkup(rich.MakeRed(r.Button.GetLabel()))
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
