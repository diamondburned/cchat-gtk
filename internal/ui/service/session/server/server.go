package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/loading"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 24
const IconSize = 18

type Controller interface {
	MessageRowSelected(*Row, cchat.ServerMessage)
}

type Row struct {
	*gtk.Box
	Button *rich.ToggleButtonImage
	Server cchat.Server
	Parent breadcrumb.Breadcrumber

	ctrl Controller

	// enum 1
	message cchat.ServerMessage

	// enum 2
	children *Children
}

func NewRow(parent breadcrumb.Breadcrumber, server cchat.Server, ctrl Controller) *Row {
	button := rich.NewToggleButtonImage(server.Name())
	button.Box.SetHAlign(gtk.ALIGN_START)
	button.Image.AddProcessors(imgutil.Round(true))
	button.Image.SetSize(IconSize)
	button.SetRelief(gtk.RELIEF_NONE)
	button.Show()

	if iconer, ok := server.(cchat.Icon); ok {
		button.Image.AsyncSetIcon(iconer.Icon, "Error getting server icon URL")
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
		row.children = NewChildren(row, ctrl)
		row.children.SetServerList(server)

		box.PackStart(row.children, false, false, 0)
		primitives.AddClass(box, "server-list")

	case cchat.ServerMessage:
		row.message = server

		primitives.AddClass(box, "server-message")
	}

	button.Connect("clicked", row.onClick)

	return row
}

// Deactivate calls the disconnect function then sets the button to false. This
// function is not thread-safe.
func (row *Row) Deactivate() {
	row.Button.SetSensitive(true) // allow clicks again
	row.Button.SetActive(false)   // stop highlighting
}

func (row *Row) GetActive() bool {
	return row.Button.GetActive()
}

func (row *Row) onClick() {
	switch {
	// If the server is a message server. We're only selected if the button is
	// pressed.
	case row.message != nil && row.GetActive():
		row.Button.SetSensitive(false) // prevent clicks from deactivating
		row.ctrl.MessageRowSelected(row, row.message)

	// If the server is a list of smaller servers.
	case row.children != nil:
		row.children.SetRevealChild(!row.children.GetRevealChild())
	}
}

func (r *Row) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(r.Parent, r.Button.GetText())
}

// Children is a children server with a reference to the parent.
type Children struct {
	*gtk.Revealer
	Main *gtk.Box
	load *loading.Button // nil after init
	List cchat.ServerList

	rowctrl Controller

	Rows   []*Row
	Parent breadcrumb.Breadcrumber
}

func NewChildren(parent breadcrumb.Breadcrumber, ctrl Controller) *Children {
	load := loading.NewButton()
	load.Show()

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.Add(load)
	main.SetMarginStart(ChildrenMargin)
	main.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.Add(main)
	rev.Show()

	return &Children{
		Revealer: rev,
		Main:     main,
		load:     load,
		rowctrl:  ctrl,
		Parent:   parent,
	}
}

func (c *Children) SetServerList(list cchat.ServerList) {
	c.List = list

	go func() {
		if err := list.Servers(c); err != nil {
			log.Error(errors.Wrap(err, "Failed to get servers"))
		}
	}()
}

func (c *Children) SetServers(servers []cchat.Server) {
	gts.ExecAsync(func() {
		// Do we have the spinning circle button? If yes, remove it.
		if c.load != nil {
			c.Main.Remove(c.load)
			c.load = nil
		}

		// Save the current state.
		var oldID string
		for _, row := range c.Rows {
			if row.GetActive() {
				oldID = row.Server.ID()
				break
			}
		}

		// Update the server list.
		for _, row := range c.Rows {
			c.Main.Remove(row)
		}

		c.Rows = make([]*Row, len(servers))

		for i, server := range servers {
			row := NewRow(c, server, c.rowctrl)
			c.Rows[i] = row
			c.Main.Add(row)
		}

		// Update parent reference? Only if it's activated.
		if oldID != "" {
			for _, row := range c.Rows {
				if row.Server.ID() == oldID {
					row.Button.SetActive(true)
				}
			}
		}
	})
}

func (c *Children) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(c.Parent)
}
