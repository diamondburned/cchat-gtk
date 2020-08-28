package actions

import (
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type MenuButton struct {
	*gtk.MenuButton

	lastsig glib.SignalHandle
	lastmod *glib.MenuModel
}

func NewMenuButton() *MenuButton {
	b, _ := gtk.MenuButtonNew()
	b.SetVAlign(gtk.ALIGN_CENTER)
	b.SetSensitive(false)

	return &MenuButton{
		MenuButton: b,
	}
}

// Bind binds the given menu. The menu's prefix MUST be a constant for this
// instance of the MenuButton.
func (m *MenuButton) Bind(menu *Menu) {
	prefix, model := menu.MenuModel()

	// Insert the action group into the menu. This will only override the old
	// action group, as the prefix is a constant for this instance.
	m.MenuButton.InsertActionGroup(prefix, menu)
	// Only after we have inserted the action group can we set the model that
	// menu has. This tells Gtk to look for the menu actions inside the inserted
	// group.
	m.MenuButton.SetMenuModel(model)

	// Unbind the last handler if we have one.
	if m.lastmod != nil {
		m.lastmod.HandlerDisconnect(m.lastsig)
	}

	// Set the current model as the last one for future calls.
	if m.lastmod = model; m.lastmod != nil {
		// If we have a model, then only activate the button when we have any
		// menu items.
		m.SetSensitive(model.GetNItems() > 0)
		// Subscribe the button to menu update events.
		m.lastsig, _ = model.Connect("items-changed", func() {
			m.SetSensitive(model.GetNItems() > 0)
		})
	} else {
		// Else, don't allow the button to be clicked at all.
		m.SetSensitive(false)
	}
}
