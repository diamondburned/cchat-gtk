package gts

import (
	"bytes"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/markbates/pkger"
	"github.com/pkg/errors"
)

var cssRepos = map[string]*gtk.CssProvider{}

func getDefaultScreen() *gdk.Screen {
	d, _ := gdk.DisplayGetDefault()
	s, _ := d.GetDefaultScreen()
	return s
}

func loadProviders(screen *gdk.Screen) {
	for file, repo := range cssRepos {
		gtk.AddProviderForScreen(
			screen, repo,
			uint(gtk.STYLE_PROVIDER_PRIORITY_APPLICATION),
		)
		// mark as done
		delete(cssRepos, file)
	}
}

func LoadCSS(name, css string) {
	prov, _ := gtk.CssProviderNew()
	if err := prov.LoadFromData(css); err != nil {
		log.Error(errors.Wrap(err, "Failed to parse CSS in "+name))
		return
	}

	cssRepos[name] = prov
}

func readFile(buf *bytes.Buffer, file string) error {
	f, err := pkger.Open(file)
	if err != nil {
		return errors.Wrap(err, "Failed to load a CSS file")
	}
	defer f.Close()

	if _, err := buf.ReadFrom(f); err != nil {
		return errors.Wrap(err, "Failed to read file")
	}

	return nil
}
