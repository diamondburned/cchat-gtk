// Package config contains UI widgets and renderers for cchat's Configurator
// interface.
package config

import (
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Configurator struct {
	cchat.Service
	cchat.Configurator
}

func MenuItem(conf Configurator) menu.Item {
	return menu.SimpleItem("Configure", func() { Spawn(conf) })
}

// Restore restores the config in the background.
func Restore(conf Configurator) {
	gts.Async(func() (func(), error) {
		c, err := conf.Configuration()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get %s config", conf.ID())
		}

		file := serviceFile(conf)

		if err := config.UnmarshalFromFile(file, &c); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal %s config", conf.ID())
		}

		if err := conf.SetConfiguration(c); err != nil {
			return nil, errors.Wrapf(err, "failed to set %s config", conf.ID())
		}

		return nil, nil
	})
}

func Spawn(conf Configurator) error {
	gts.Async(func() (func(), error) {
		c, err := conf.Configuration()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get %s config", conf.ID())
		}

		file := serviceFile(conf)

		err = config.UnmarshalFromFile(file, &c)
		err = errors.Wrapf(err, "failed to unmarshal %s config", conf.ID())

		return func() {
			spawn(conf, c, func(finalized bool) error {
				if err := conf.SetConfiguration(c); err != nil {
					return err
				}

				if finalized {
					gts.Async(func() (func(), error) {
						return nil, config.MarshalToFile(file, c)
					})
				}

				return nil
			})
		}, err
	})

	return nil
}

func serviceFile(conf Configurator) string {
	return fmt.Sprintf("services/%s.json", conf.ID())
}

func spawn(c Configurator, conf map[string]string, apply func(final bool) error) {
	container := newContainer(conf, func() error { return apply(false) })
	container.Grid.SetVAlign(gtk.ALIGN_START)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Add(container.Grid)
	sw.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, vmargin)
	b.SetMarginTop(vmargin)
	b.SetMarginBottom(vmargin)
	b.SetMarginStart(hmargin)
	b.SetMarginEnd(hmargin)
	b.PackStart(sw, true, true, 0)
	b.PackStart(container.ErrHeader, false, false, 0)
	b.Show()

	h, _ := gtk.HeaderBarNew()
	h.SetTitle("Configure " + c.ID())
	h.SetShowCloseButton(true)
	h.Show()

	var state rich.NameContainer
	state.OnUpdate(func() {
		h.SetTitle("Configure " + state.String())
	})

	d, _ := gts.NewEmptyModalDialog()
	d.SetDefaultSize(400, 300)
	d.Add(b)
	d.SetTitlebar(h)

	// Bind the title.
	state.BindNamer(d, "response", c)
	// Bind the updater.
	d.Connect("response", func(*gtk.Dialog) { apply(true) })

	d.Show()
}
