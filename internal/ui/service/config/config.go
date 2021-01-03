// Package config contains UI widgets and renderers for cchat's Configurator
// interface.
package config

import (
	"fmt"
	"hash/fnv"
	"io"
	"strconv"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat/text"
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
			return nil, errors.Wrapf(err, "failed to get %s config", conf.Name())
		}

		file := serviceFile(conf)

		if err := config.UnmarshalFromFile(file, &c); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal %s config", conf.Name())
		}

		if err := conf.SetConfiguration(c); err != nil {
			return nil, errors.Wrapf(err, "failed to set %s config", conf.Name())
		}

		return nil, nil
	})
}

func Spawn(conf Configurator) error {
	gts.Async(func() (func(), error) {
		c, err := conf.Configuration()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get %s config", conf.Name())
		}

		file := serviceFile(conf)

		err = config.UnmarshalFromFile(file, &c)
		err = errors.Wrapf(err, "failed to unmarshal %s config", conf.Name())

		return func() {
			spawn(conf.Name().String(), c, func(finalized bool) error {
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
	return fmt.Sprintf("service-%s.json", dumbHash(conf.Name()))
}

func dumbHash(name text.Rich) string {
	hash := fnv.New32a()
	io.WriteString(hash, name.String())
	return strconv.FormatUint(uint64(hash.Sum32()), 36)
}

func spawn(name string, conf map[string]string, apply func(final bool) error) {
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

	var title = "Configure " + name

	h, _ := gtk.HeaderBarNew()
	h.SetTitle(title)
	h.SetShowCloseButton(true)
	h.Show()

	d, _ := gts.NewEmptyModalDialog()
	d.SetDefaultSize(400, 300)
	d.Add(b)
	d.SetTitle(title)
	d.SetTitlebar(h)

	d.Connect("destroy", func(*gtk.Dialog) { apply(true) })

	d.Show()
}
