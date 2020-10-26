package actions

import (
	"fmt"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type ActionGroupInserter interface {
	InsertActionGroup(prefix string, action glib.IActionGroup)
}

var _ ActionGroupInserter = (*gtk.Widget)(nil)

type Menu struct {
	*Stateful
	menu   *glib.Menu
	prefix string
}

func NewMenu(prefix string) *Menu {
	return &Menu{
		Stateful: NewStateful(), // actiongroup and menu not linked
		menu:     glib.MenuNew(),
		prefix:   prefix,
	}
}

func (m *Menu) Prefix() string {
	return m.prefix
}

func (m *Menu) MenuModel() (string, *glib.MenuModel) {
	return m.prefix, &m.menu.MenuModel
}

func (m *Menu) InsertActionGroup(w ActionGroupInserter) {
	w.InsertActionGroup(m.prefix, m)
}

// Popup pops up the menu popover. It does not pop up anything if there are no
// menu items.
func (m *Menu) Popup(relative gtk.IWidget) {
	p := m.popover(relative)
	if p == nil || m.Len() == 0 {
		return
	}

	p.Popup()
}

func (m *Menu) popover(relative gtk.IWidget) *gtk.Popover {
	_, model := m.MenuModel()

	p, _ := gtk.PopoverNewFromModel(relative, model)
	p.SetPosition(gtk.POS_RIGHT)

	return p
}

func (m *Menu) Reset() {
	m.menu.RemoveAll()
	m.Stateful.Reset()
}

func (m *Menu) AddAction(label string, call func()) {
	m.Stateful.AddAction(label, call)
	m.menu.Append(label, fmt.Sprintf("%s.%s", m.prefix, ActionName(label)))
}

func (m *Menu) RemoveAction(label string) {
	var labels = m.Stateful.labels

	for i, l := range labels {
		if l == label {
			labels = append(labels[:i], labels[:i+1]...)
			m.menu.Remove(i)

			m.Stateful.labels = labels
			m.Stateful.group.RemoveAction(ActionName(label))

			return
		}
	}
}
