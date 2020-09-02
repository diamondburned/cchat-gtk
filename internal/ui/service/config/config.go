// Package config contains UI widgets and renderers for cchat's Configurator
// interface.
package config

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/gotk3/gotk3/gtk"
)

type Configurator interface {
	cchat.Service
	cchat.Configurator
}

func MenuItem(conf Configurator) menu.Item {
	return menu.SimpleItem("Configure", func() {
		SpawnConfigurator(conf)
	})
}

func SpawnConfigurator(conf Configurator) error {
	c, err := conf.Configuration()
	if err != nil {
		return err
	}

	Spawn(conf.Name().Content, c, func() error {
		return conf.SetConfiguration(c)
	})

	return nil
}

func Spawn(name string, conf map[string]string, apply func() error) {
	container := newContainer(conf, apply)
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
	d.Show()
}
