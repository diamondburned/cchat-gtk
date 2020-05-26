package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Row struct {
	*gtk.Box
	Button *gtk.Button
	Server cchat.Server

	// enum 1
	clicked func(*Row)
	message cchat.ServerMessage

	// enum 2
	children *Children
}

func New(server cchat.Server) *Row {
	name, err := server.Name()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to get the server name"))
		name = "no name"
	}

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()

	button, _ := gtk.ButtonNew()
	button.Show()
	button.SetRelief(gtk.RELIEF_NONE)
	button.SetLabel(name)

	// TODO: images

	var row = &Row{
		Box:    box,
		Button: button,
		Server: server,
	}
	button.Connect("clicked", row.onClick)

	switch server := server.(type) {
	case cchat.ServerList:
		row.children = NewChildren(server)
	case cchat.ServerMessage:
		row.message = server
	}

	return row
}

// SetOnClick sets the callback when the server is clicked. This only works if
// the passed in server implements ServerMessage.
func (row *Row) SetOnClick(clicked func(*Row)) {
	if row.message != nil {
		row.clicked = clicked
	}
}

func (row *Row) onClick() {
	switch {
	case row.message != nil:
		row.clicked(row)
	case row.children != nil:
		row.children.SetRevealChild(!row.children.GetRevealChild())
	}
}

type Children struct {
	*gtk.Revealer
	Main *gtk.Box
	Rows []*Row
	List cchat.ServerList
}

func NewChildren(list cchat.ServerList) *Children {
	rev, _ := gtk.RevealerNew()
	rev.Show()
	rev.SetRevealChild(false)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	main.Show()

	children := &Children{
		Revealer: rev,
		Main:     main,
		List:     list,
	}

	if err := list.Servers(children); err != nil {
		log.Error(errors.Wrap(err, "Failed to get servers"))
	}

	return children
}

func (c *Children) SetServers(servers []cchat.Server) {
	gts.ExecAsync(func() {
		for _, row := range c.Rows {
			c.Main.Remove(row)
		}

		c.Rows = make([]*Row, len(servers))

		for i, server := range servers {
			row := New(server)
			c.Rows[i] = row
			c.Main.Add(row)
		}
	})
}
