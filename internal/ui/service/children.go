package service

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/gotk3/gotk3/gtk"
)

type children struct {
	*gtk.Box
	Sessions map[string]*session.Row
}

func newChildren() *children {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	return &children{box, map[string]*session.Row{}}
}

func (c *children) addSessionRow(id string, row *session.Row) {
	c.Sessions[id] = row
	c.Box.Add(row)
}

func (c *children) removeSessionRow(id string) {
	if row, ok := c.Sessions[id]; ok {
		delete(c.Sessions, id)
		c.Box.Remove(row)
	}
}
