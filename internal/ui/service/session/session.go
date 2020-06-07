package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const IconSize = 32

// Controller extends server.RowController to add session.
type Controller interface {
	MessageRowSelected(*Row, *server.Row, cchat.ServerMessage)
	RestoreSession(*Row, cchat.SessionRestorer) // async
}

type Row struct {
	*gtk.Box
	Button  *rich.ToggleButtonImage
	Session cchat.Session

	Servers *server.Children

	ctrl   Controller
	parent breadcrumb.Breadcrumber
}

func New(parent breadcrumb.Breadcrumber, ses cchat.Session, ctrl Controller) *Row {
	row := new(parent, ctrl)
	row.SetSession(ses)
	return row
}

func NewLoading(parent breadcrumb.Breadcrumber, name string, ctrl Controller) *Row {
	row := new(parent, ctrl)
	row.Button.SetLabelUnsafe(text.Rich{Content: name})
	row.Button.Image.SetPlaceholderIcon("content-loading-symbolic", IconSize)
	row.SetSensitive(false)

	return row
}

func new(parent breadcrumb.Breadcrumber, ctrl Controller) *Row {
	row := &Row{
		ctrl:   ctrl,
		parent: parent,
	}

	row.Button = rich.NewToggleButtonImage(text.Rich{})
	row.Button.Box.SetHAlign(gtk.ALIGN_START)
	row.Button.Image.AddProcessors(imgutil.Round(true))
	// Set the loading icon.
	row.Button.SetRelief(gtk.RELIEF_NONE)
	// On click, toggle reveal.
	row.Button.Connect("clicked", func() {
		revealed := !row.Servers.GetRevealChild()
		row.Servers.SetRevealChild(revealed)
		row.Button.SetActive(revealed)
	})
	row.Button.Show()

	row.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	row.Box.SetMarginStart(server.ChildrenMargin)
	row.Box.PackStart(row.Button, false, false, 0)
	row.Box.Show()

	primitives.AddClass(row.Box, "session")

	return row
}

// KeyringSession returns a keyring session, or nil if the session cannot be
// saved. This function is not cached, as I'd rather not keep the map in memory.
func (r *Row) KeyringSession() *keyring.Session {
	// Is the session saveable?
	saver, ok := r.Session.(cchat.SessionSaver)
	if !ok {
		return nil
	}

	ks := keyring.Session{
		ID:   r.Session.ID(),
		Name: r.Button.GetText(),
	}

	s, err := saver.Save()
	if err != nil {
		log.Error(errors.Wrapf(err, "Failed to save session ID %s (%s)", ks.ID, ks.Name))
		return nil
	}
	ks.Data = s

	return &ks
}

func (r *Row) SetSession(ses cchat.Session) {
	r.Session = ses
	r.Servers = server.NewChildren(r, ses, r)
	r.Button.Image.SetPlaceholderIcon("user-available-symbolic", IconSize)
	r.Box.PackStart(r.Servers, false, false, 0)
	r.SetSensitive(true)

	// Set the session's name to the button.
	r.Button.Try(ses, "session")
}

func (r *Row) SetFailed(err error) {
	r.SetTooltipText(err.Error())
	// TODO: setting the label directly here is kind of shitty, as it screws up
	// the getter. Fix?
	r.Button.Label.SetMarkup(rich.MakeRed(r.Button.GetLabel()))
}

func (r *Row) MessageRowSelected(server *server.Row, smsg cchat.ServerMessage) {
	r.ctrl.MessageRowSelected(r, server, smsg)
}

func (r *Row) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(r.parent, r.Button.GetLabel().Content)
}
