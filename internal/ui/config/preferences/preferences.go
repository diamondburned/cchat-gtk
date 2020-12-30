package preferences

import (
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/dialog"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Dialog struct {
	*dialog.Dialog

	switcher *gtk.StackSwitcher
	stack    *gtk.Stack
}

func NewDialog() *Dialog {
	stack, _ := gtk.StackNew()
	stack.SetMarginTop(8)
	stack.SetMarginBottom(8)
	stack.SetMarginStart(16)
	stack.SetMarginEnd(16)
	stack.Show()

	switcher, _ := gtk.StackSwitcherNew()
	switcher.SetStack(stack)
	switcher.Show()

	h, _ := gtk.HeaderBarNew()
	h.SetShowCloseButton(true)
	h.SetCustomTitle(switcher)
	h.Show()

	d := dialog.NewCSD(stack, h)
	d.SetDefaultSize(400, 300)
	d.SetTitle("Preferences")

	return &Dialog{
		Dialog:   d,
		stack:    stack,
		switcher: switcher,
	}
}

func Section(entries []config.Entry) *gtk.Grid {
	var grid, _ = gtk.GridNew()

	for i, entry := range entries {
		l, _ := gtk.LabelNew(entry.Name)
		l.SetHExpand(true)
		l.SetXAlign(0)
		l.Show()

		grid.Attach(l, 0, i, 1, 1)
		grid.Attach(entry.Value.Construct(), 1, i, 1, 1)
	}

	grid.SetRowSpacing(4)
	grid.SetColumnSpacing(8)
	grid.Show()

	primitives.AddClass(grid, "config")

	return grid
}

func NewPreferenceDialog() *Dialog {
	var dialog = NewDialog()

	for i, section := range config.Sections() {
		grid := Section(section)
		name := config.Section(i).String()

		dialog.stack.AddTitled(grid, name, name)
	}

	return dialog
}

func SpawnPreferenceDialog() {
	p := NewPreferenceDialog()
	p.Connect("destroy", func(interface{}) {
		// On close, save the settings.
		if err := config.Save(); err != nil {
			log.Error(errors.Wrap(err, "Failed to save settings"))
		}
	})
	p.Show()
}
