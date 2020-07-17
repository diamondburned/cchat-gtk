package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/loading"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/traverse"
	"github.com/gotk3/gotk3/gtk"
)

type Controller interface {
	RowSelected(*ServerRow, cchat.ServerMessage)
}

// Children is a children server with a reference to the parent.
type Children struct {
	*gtk.Box
	load *loading.Button // only not nil while loading

	Rows []*ServerRow

	Parent  traverse.Breadcrumber
	rowctrl Controller
}

// reserved
var childrenCSS = primitives.PrepareClassCSS("server-children", `
	.server-children {
		margin: 0;
		margin-top: 3px;
		border-radius: 0;
	}
`)

func NewChildren(p traverse.Breadcrumber, ctrl Controller) *Children {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.SetMarginStart(ChildrenMargin)
	childrenCSS(main)

	return &Children{
		Box:     main,
		Parent:  p,
		rowctrl: ctrl,
	}
}

// setLoading shows the loading circle as a list child.
func (c *Children) setLoading() {
	// Exit if we're already loading.
	if c.load != nil {
		return
	}

	// Clear everything.
	c.Reset()

	// Set the loading circle and stuff.
	c.load = loading.NewButton()
	c.load.Show()
	c.Box.Add(c.load)
}

func (c *Children) Reset() {
	// Remove old servers from the list.
	for _, row := range c.Rows {
		c.Box.Remove(row)
	}

	// Wipe the list empty.
	c.Rows = nil
}

// setNotLoading removes the loading circle, if any. This is not in Reset()
// anymore, since the backend may not necessarily call SetServers.
func (c *Children) setNotLoading() {
	// Do we have the spinning circle button? If yes, remove it.
	if c.load != nil {
		// Stop the loading mode. The reset function should do everything for us.
		c.Box.Remove(c.load)
		c.load = nil
	}
}

// SetServers is reserved for cchat.ServersContainer.
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

		// Reset before inserting new servers.
		c.Reset()

		c.Rows = make([]*ServerRow, len(servers))

		for i, server := range servers {
			row := NewServerRow(c, server, c.rowctrl)
			row.Show()
			// row.SetFocusHAdjustment(c.GetFocusHAdjustment()) // inherit
			// row.SetFocusVAdjustment(c.GetFocusVAdjustment())

			c.Rows[i] = row
			c.Box.Add(row)
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

func (c *Children) Breadcrumb() traverse.Breadcrumb {
	return traverse.TryBreadcrumb(c.Parent)
}
