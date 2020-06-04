package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Row struct {
	*gtk.Box
	Button *rich.ToggleButtonImage
	Server cchat.Server
	Parent *Children

	ctrl Controller

	// enum 1
	message cchat.ServerMessage

	// enum 2
	children *Children
}

func NewRow(parent *Children, server cchat.Server, ctrl Controller) *Row {
	button := rich.NewToggleButtonImage(text.Rich{}, "")
	button.Box.SetHAlign(gtk.ALIGN_START)
	button.SetRelief(gtk.RELIEF_NONE)
	button.Show()

	if err := server.Name(button); err != nil {
		log.Error(errors.Wrap(err, "Failed to get the server name"))
		button.SetLabel(text.Rich{Content: "Unknown"})
	}

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(button, false, false, 0)
	box.Show()

	primitives.AddClass(box, "server")

	// TODO: images

	var row = &Row{
		Box:    box,
		Button: button,
		Server: server,
		Parent: parent,
		ctrl:   ctrl,
	}

	switch server := server.(type) {
	case cchat.ServerList:
		row.children = NewChildren(row, server, ctrl)
		box.PackStart(row.children, false, false, 0)

		primitives.AddClass(box, "server-list")

	case cchat.ServerMessage:
		row.message = server

		primitives.AddClass(box, "server-message")
	}

	button.Connect("clicked", row.onClick)

	return row
}

func (row *Row) GetActive() bool {
	return row.Button.GetActive()
}

func (row *Row) onClick() {
	switch {

	// If the server is a message server. We're only selected if the button is
	// pressed.
	case row.message != nil && row.GetActive():
		row.ctrl.MessageRowSelected(row, row.message)

	// If the server is a list of smaller servers.
	case row.children != nil:
		row.children.SetRevealChild(!row.children.GetRevealChild())
	}
}

func (r *Row) Breadcrumb() string {
	var label = r.Button.GetLabel().Content

	// Does the row have a parent?
	if r.Parent != nil {
		return r.Parent.ParentRow.Breadcrumb() + "/" + label
	}

	return label
}
