package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/loading"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/savepath"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/gotk3/gotk3/gtk"
)

type Controller interface {
	MessengerSelected(*ServerRow)
}

// Children is a children server with a reference to the parent. By default, a
// children will contain hollow rows. They are rows that do not yet have any
// widgets. This changes as soon as Row's Load is called.
type Children struct {
	*gtk.Box

	load    *loading.Button // only not nil while loading
	loading bool

	Rows []*ServerRow

	Parent  traverse.Breadcrumber
	rowctrl Controller

	// Unreadable state for children rows to use. The parent row that has this
	// Children will bind a handler to this.
	traverse.Unreadable
}

var childrenCSS = primitives.PrepareClassCSS("server-children", `
	.server-children {
		margin: 0;
		margin-top: 3px;
		border-radius: 0;
	}
`)

// NewHollowChildren creates a hollow children, which is a children without any
// widgets.
func NewHollowChildren(p traverse.Breadcrumber, ctrl Controller) *Children {
	return &Children{
		Parent:  p,
		rowctrl: ctrl,
	}
}

// NewChildren creates a hollow children then immediately unhollows it.
func NewChildren(p traverse.Breadcrumber, ctrl Controller) *Children {
	c := NewHollowChildren(p, ctrl)
	c.Init()
	return c
}

func (c *Children) IsHollow() bool {
	return c.Box == nil
}

// Init ensures that the children container is not hollow. It does nothing after
// the first call. It does not actually populate the list with widgets. This is
// done for lazy loading. To load everything, call LoadAll after this.
//
// Nothing but ServerRow should call this method.
func (c *Children) Init() {
	if c.IsHollow() {
		c.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		c.Box.SetMarginStart(ChildrenMargin)
		c.Box.SetHExpand(true)
		childrenCSS(c.Box)

		// Check if we're still loading. This is effectively restoring the
		// state that was set before we had widgets.
		if c.loading {
			c.setLoading()
		} else {
			c.setNotLoading()
		}
	}
}

// Reset ensures that the children container is no longer hollow, then reset all
// states.
func (c *Children) Reset() {
	// If the children container isn't hollow, then we have to remove the known
	// rows from the container box.
	if c.Box != nil {
		// Remove old servers from the list.
		for _, row := range c.Rows {
			if row.IsHollow() {
				continue
			}
			c.Box.Remove(row)
		}
	}

	// Wipe the list empty.
	c.Rows = nil
}

// setLoading shows the loading circle as a list child. If hollow, this function
// will only update the state.
func (c *Children) setLoading() {
	c.loading = true

	// Don't do the rest if we're still hollow.
	if c.IsHollow() {
		return
	}

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

// setNotLoading removes the loading circle, if any. This is not in Reset()
// anymore, since the backend may not necessarily call SetServers.
func (c *Children) setNotLoading() {
	c.loading = false

	// Don't call the rest if we're still hollow.
	if c.IsHollow() {
		return
	}

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
		// Save the current state (if any) if the children container is not
		// hollow.
		if !c.IsHollow() {
			restore := c.saveSelectedRow()
			defer restore()
		}

		// Reset before inserting new servers.
		c.Reset()

		// Insert hollow servers.
		c.Rows = make([]*ServerRow, len(servers))
		for i, server := range servers {
			c.Rows[i] = NewHollowServer(c, server, c.rowctrl)
		}

		// We should not unhollow everything here, but rather on uncollapse.
		// Since the root node is always unhollow, calls to this function will
		// pass the hollow test and unhollow its children nodes. That should not
		// happen.
	})
}

func (c *Children) findID(id cchat.ID) (int, *ServerRow) {
	for i, row := range c.Rows {
		if row.Server.ID() == id {
			return i, row
		}
	}
	return -1, nil
}

func (c *Children) insertAt(row *ServerRow, i int) {
	c.Rows = append(c.Rows[:i], append([]*ServerRow{row}, c.Rows[i:]...)...)

	if !c.IsHollow() {
		c.Box.Add(row)
		c.Box.ReorderChild(row, i)
	}
}

func (c *Children) UpdateServer(update cchat.ServerUpdate) {
	gts.ExecAsync(func() {
		prevID, replace := update.PreviousID()

		// TODO: I don't think this code unhollows a new server.
		var newServer = NewHollowServer(c, update, c.rowctrl)
		var i, oldRow = c.findID(prevID)

		// If we're appending a new row, then replace is false.
		if !replace {
			// Increment the old row's index so we know where to insert.
			c.insertAt(newServer, i+1)
			return
		}

		// Only update the server if the old row was found.
		if oldRow == nil {
			return
		}

		c.Rows[i] = newServer

		if !c.IsHollow() {
			// Update the UI as well.
			// TODO: check if this reorder is correct.
			c.Box.Remove(oldRow)
			c.Box.Add(newServer)
			c.Box.ReorderChild(newServer, i)
		}
	})
}

// LoadAll forces all children rows to be unhollowed (initialized). It does
// NOT check if the children container itself is hollow.
func (c *Children) LoadAll() {
	AssertUnhollow(c)

	for _, row := range c.Rows {
		if row.IsHollow() {
			row.Init() // this is the alloc-heavy method
			row.Show()
			c.Box.Add(row)
		}

		// Restore expansion if possible.
		savepath.Restore(row, row.Button)
	}

	// Check if we have icons.
	var hasIcon bool

	for _, row := range c.Rows {
		if row.HasIcon() {
			hasIcon = true
			break
		}
	}

	// If we have an icon, then show all other possibly empty icons. HdyAvatar
	// will generate a placeholder.
	if hasIcon {
		for _, row := range c.Rows {
			row.UseEmptyIcon()
		}
	}
}

// saveSelectedRow saves the current selected row and returns a callback that
// restores the selection.
func (c *Children) saveSelectedRow() (restore func()) {
	// Save the current state.
	var oldID string
	for _, row := range c.Rows {
		if row.GetActive() {
			oldID = row.Server.ID()
			break
		}
	}

	return func() {
		if oldID != "" {
			for _, row := range c.Rows {
				if row.Server.ID() == oldID {
					row.Init()
					row.Button.SetActive(true)
				}
			}
		}

		// TODO Update parent reference? Only if it's activated.
	}
}

func (c *Children) ParentBreadcrumb() traverse.Breadcrumber {
	return c.Parent
}
