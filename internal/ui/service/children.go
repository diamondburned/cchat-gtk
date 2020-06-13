package service

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/gotk3/gotk3/gtk"
)

type children struct {
	*gtk.Box
	sessions map[string]*session.Row
}

func newChildren() *children {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	return &children{box, map[string]*session.Row{}}
}

func (c *children) Sessions() []*session.Row {
	// We already know the size beforehand. Allocate it wisely.
	var rows = make([]*session.Row, 0, len(c.sessions))

	// Loop over widget children.
	primitives.EachChildren(c.Box, func(i int, v interface{}) bool {
		var id = primitives.GetName(v.(primitives.Namer))

		if row, ok := c.sessions[id]; ok {
			rows = append(rows, row)
		}

		return false
	})

	return rows
}

func (c *children) AddSessionRow(id string, row *session.Row) {
	c.sessions[id] = row
	c.Box.Add(row)

	// Bind the mover.
	row.BindMover(id)

	// Assert that a name can be obtained.
	namer := primitives.Namer(row)
	namer.SetName(id) // set ID here, get it in Move
}

func (c *children) RemoveSessionRow(sessionID string) bool {
	row, ok := c.sessions[sessionID]
	if ok {
		delete(c.sessions, sessionID)
		c.Box.Remove(row)
	}
	return ok
}

func (c *children) MoveSession(id, movingID string) {
	// Get the widget of the row that is moving.
	var moving = c.sessions[movingID]

	// Find the current position of the row that we're moving the other one
	// underneath of.
	var rowix = -1

	primitives.EachChildren(c.Box, func(i int, v interface{}) bool {
		// The obtained name will be the ID set in AddSessionRow.
		if primitives.GetName(v.(primitives.Namer)) == id {
			rowix = i
			return true
		}

		return false
	})

	// Reorder the box.
	c.Box.ReorderChild(moving, rowix)
}
