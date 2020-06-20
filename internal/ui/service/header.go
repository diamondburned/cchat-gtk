package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

const IconSize = 32

type header struct {
	*gtk.ToggleButton // no rich text here but it's left aligned

	box   *gtk.Box
	label *rich.Label
	icon  *rich.Icon
	Add   *gtk.Button

	Menu *gtk.Menu
}

func newHeader(svc cchat.Service) *header {
	i := rich.NewIcon(0)
	i.AddProcessors(imgutil.Round(true))
	i.SetPlaceholderIcon("folder-remote-symbolic", IconSize)
	i.Show()

	if iconer, ok := svc.(cchat.Icon); ok {
		i.AsyncSetIconer(iconer, "Error getting session logo")
	}

	l := rich.NewLabel(svc.Name())
	l.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(i, false, false, 0)
	box.PackStart(l, true, true, 5)
	box.SetMarginEnd(IconSize) // spare space for the add button
	box.Show()

	add, _ := gtk.ButtonNewFromIconName("list-add-symbolic", gtk.ICON_SIZE_BUTTON)
	add.SetRelief(gtk.RELIEF_NONE)
	add.SetSizeRequest(IconSize, IconSize)
	add.SetHAlign(gtk.ALIGN_END)
	add.Show()

	// Do jank stuff to overlay the add button on top of our button.
	overlay, _ := gtk.OverlayNew()
	overlay.Add(box)
	overlay.AddOverlay(add)
	overlay.Show()

	reveal, _ := gtk.ToggleButtonNew()
	reveal.Add(overlay)
	reveal.SetRelief(gtk.RELIEF_NONE)
	reveal.SetMode(true)
	reveal.Show()

	// Spawn the menu on right click.
	menu, _ := gtk.MenuNew()
	primitives.BindMenu(reveal, menu)

	return &header{reveal, box, l, i, add, menu}
}

func (h *header) GetText() string {
	return h.label.GetText()
}
