package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

const IconSize = 32

// Controller extends server.RowController to add session.
type Controller interface {
	MessageRowSelected(*Row, *server.Row, cchat.ServerMessage)
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
	row := &Row{
		Session: ses,
		ctrl:    ctrl,
		parent:  parent,
	}
	row.Servers = server.NewChildren(row, ses, row)

	row.Button = rich.NewToggleButtonImage(text.Rich{})
	row.Button.Box.SetHAlign(gtk.ALIGN_START)
	row.Button.Image.SetPlaceholderIcon("user-available-symbolic", IconSize)
	row.Button.SetRelief(gtk.RELIEF_NONE)
	// On click, toggle reveal.
	row.Button.Connect("clicked", func() {
		revealed := !row.Servers.GetRevealChild()
		row.Servers.SetRevealChild(revealed)
		row.Button.SetActive(revealed)
	})
	row.Button.Show()
	row.Button.Try(ses, "session")

	row.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	row.Box.SetMarginStart(server.ChildrenMargin)
	row.Box.PackStart(row.Button, false, false, 0)
	row.Box.PackStart(row.Servers, false, false, 0)
	row.Box.Show()

	primitives.AddClass(row.Box, "session")

	return row
}

func (r *Row) MessageRowSelected(server *server.Row, smsg cchat.ServerMessage) {
	r.ctrl.MessageRowSelected(r, server, smsg)
}

func (r *Row) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(r.parent, r.Button.GetLabel().Content)
}
