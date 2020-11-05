package throttler

import (
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const TPS = 24 // tps

type State struct {
	throttling bool
	ticker     <-chan time.Time
	settings   *gtk.Settings
}

type Connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func Bind(app *gtk.Application) *State {
	var settings, _ = gtk.SettingsGetDefault()
	var s = State{
		settings: settings,
		ticker:   time.Tick(time.Second / TPS),
	}

	app.Connect("window-added", func(app *gtk.Application, w *gtk.Window) {
		s.Connect(w)
	})

	return &s
}

func (s *State) Connect(c Connector) {
	c.Connect("focus-out-event", s.Start)
	c.Connect("focus-in-event", s.Stop)
}

func (s *State) Start() {
	if s.throttling {
		return
	}

	s.throttling = true
	s.settings.SetProperty("gtk-enable-animations", false)

	glib.IdleAdd(func() bool {
		// Throttle.
		<-s.ticker

		// If we're no longer throttling, then stop the ticker and remove this
		// callback from the loop.
		if !s.throttling {
			return false
		}

		// Keep calling this same callback.
		return true
	})
}

func (s *State) Stop() {
	if !s.throttling {
		return
	}

	s.throttling = false
	s.settings.SetProperty("gtk-enable-animations", true)
}
