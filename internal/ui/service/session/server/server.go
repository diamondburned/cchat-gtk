package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 24

type RowController interface {
	MessageRowSelected(*Row, cchat.ServerMessage)
}

type Row struct {
	*gtk.Box
	Button *gtk.Button
	Server cchat.Server

	ctrl RowController

	// enum 1
	message cchat.ServerMessage

	// enum 2
	children *Children
}

func New(server cchat.Server, ctrl RowController) *Row {
	name, err := server.Name()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to get the server name"))
		name = "no name"
	}

	button, _ := gtk.ButtonNewWithLabel(name)
	primitives.BinLeftAlignLabel(button)

	button.SetRelief(gtk.RELIEF_NONE)
	button.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(button, false, false, 0)
	box.Show()

	primitives.AddClass(box, "server")

	// TODO: images

	var row = &Row{
		Box:    box,
		Button: button,
		Server: server,
		ctrl:   ctrl,
	}
	button.Connect("clicked", row.onClick)

	switch server := server.(type) {
	case cchat.ServerList:
		row.children = NewChildren(server, ctrl)
		box.PackStart(row.children, false, false, 0)

		primitives.AddClass(box, "server-list")

	case cchat.ServerMessage:
		row.message = server

		primitives.AddClass(box, "server-message")
	}

	return row
}

func (row *Row) onClick() {
	switch {
	case row.message != nil:
		row.ctrl.MessageRowSelected(row, row.message)
	case row.children != nil:
		row.children.SetRevealChild(!row.children.GetRevealChild())
	}
}

type Children struct {
	*gtk.Revealer
	Main *gtk.Box
	List cchat.ServerList

	Rows    []*Row
	rowctrl RowController
}

func NewChildren(list cchat.ServerList, ctrl RowController) *Children {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetMarginStart(ChildrenMargin)
	main.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.Add(main)
	rev.Show()

	children := &Children{
		Revealer: rev,
		Main:     main,
		List:     list,
		rowctrl:  ctrl,
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
			row := New(server, c.rowctrl)
			c.Rows[i] = row
			c.Main.Add(row)
		}
	})
}
