package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 24

type Controller interface {
	MessageRowSelected(*Row, cchat.ServerMessage)
}

// Children is a children server with a reference to the parent.
type Children struct {
	*gtk.Revealer
	Main *gtk.Box
	List cchat.ServerList

	rowctrl Controller

	Rows      []*Row
	ParentRow *Row
}

func NewChildren(parent *Row, list cchat.ServerList, ctrl Controller) *Children {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetMarginStart(ChildrenMargin)
	main.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.Add(main)
	rev.Show()

	children := &Children{
		Revealer:  rev,
		Main:      main,
		List:      list,
		rowctrl:   ctrl,
		ParentRow: parent,
	}

	if err := list.Servers(children); err != nil {
		log.Error(errors.Wrap(err, "Failed to get servers"))
	}

	return children
}

func (c *Children) SetServers(servers []cchat.Server) {
	gts.ExecAsync(func() {
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
