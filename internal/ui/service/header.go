package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const IconSize = 32

type header struct {
	*gtk.Box
	reveal *rich.ToggleButtonImage // no rich text here but it's left aligned
	add    *gtk.Button

	Menu *gtk.Menu
}

func newHeader(svc cchat.Service) *header {
	reveal := rich.NewToggleButtonImage(svc.Name())
	reveal.Box.SetHAlign(gtk.ALIGN_START)
	reveal.Image.AddProcessors(imgutil.Round(true))
	reveal.Image.SetPlaceholderIcon("folder-remote-symbolic", IconSize)
	reveal.SetRelief(gtk.RELIEF_NONE)
	reveal.SetMode(true)
	reveal.Show()

	add, _ := gtk.ButtonNewFromIconName("list-add-symbolic", gtk.ICON_SIZE_BUTTON)
	add.SetRelief(gtk.RELIEF_NONE)
	add.SetSizeRequest(IconSize, IconSize)
	add.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(reveal, true, true, 0)
	box.PackStart(add, false, false, 0)
	box.Show()

	if iconer, ok := svc.(cchat.Icon); ok {
		if err := iconer.Icon(reveal); err != nil {
			log.Error(errors.Wrap(err, "Error getting session logo"))
		}
	}

	// Spawn the menu on right click.
	menu, _ := gtk.MenuNew()
	primitives.BindMenu(reveal, menu)

	return &header{box, reveal, add, menu}
}
