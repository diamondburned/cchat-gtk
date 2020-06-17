package menu

import (
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// LazyMenu is a menu with lazy-loaded capabilities.
type LazyMenu struct {
	items []Item
}

func NewLazyMenu(bindTo primitives.Connector) *LazyMenu {
	l := &LazyMenu{}
	bindTo.Connect("button-press-event", l.popup)
	return l
}

func (m *LazyMenu) SetItems(items []Item) {
	m.items = items
}

func (m *LazyMenu) AddItems(items ...Item) {
	m.items = append(m.items, items...)
}

func (m *LazyMenu) Reset() {
	m.items = nil
}

func (m *LazyMenu) popup(w gtk.IWidget, ev *gdk.Event) {
	// Is this a right click? Exit if not.
	if !gts.EventIsRightClick(ev) {
		return
	}

	// Do nothing if there are no menu items.
	if len(m.items) == 0 {
		return
	}

	var menu, _ = gtk.MenuNew()

	for _, item := range m.items {
		mb, _ := gtk.MenuItemNewWithLabel(item.Name)
		mb.Connect("activate", item.Func)
		mb.Show()

		if item.Extra != nil {
			item.Extra(mb)
		}

		menu.Append(mb)
	}

	menu.PopupAtPointer(ev)
}

type Item struct {
	Name  string
	Func  func()
	Extra func(*gtk.MenuItem)
}

func SimpleItem(name string, fn func()) Item {
	return Item{Name: name, Func: fn}
}
