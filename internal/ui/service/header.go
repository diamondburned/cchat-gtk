package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/buttonoverlay"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/gotk3/gotk3/gtk"
)

const IconSize = 32

type header struct {
	*rich.ToggleButtonImage
	Add *gtk.Button

	Menu *menu.LazyMenu
}

func newHeader(svc cchat.Service) *header {
	b := rich.NewToggleButtonImage(svc.Name())
	b.Image.SetPlaceholderIcon("folder-remote-symbolic", IconSize)
	b.SetRelief(gtk.RELIEF_NONE)
	b.SetMode(true)
	b.Show()

	if iconer, ok := svc.(cchat.Icon); ok {
		b.Image.AsyncSetIconer(iconer, "Error getting session logo")
	}

	add, _ := gtk.ButtonNewFromIconName("list-add-symbolic", gtk.ICON_SIZE_BUTTON)
	add.Show()

	// Add the button overlay into the main button.
	buttonoverlay.Take(b, add, IconSize)

	// Construct a menu and its items.
	var menu = menu.NewLazyMenu(b)
	if configurator, ok := svc.(config.Configurator); ok {
		menu.AddItems(config.MenuItem(configurator))
	}

	return &header{b, add, menu}
}
