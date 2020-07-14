package throttler

import (
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const TPS = 15 // tps

type State struct {
	throttling bool
	ticker     <-chan time.Time
	settings   *gtk.Settings
}

type Connector interface {
	gtk.IWidget
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func Bind(evc Connector) *State {
	var settings, _ = gtk.SettingsGetDefault()
	var s = State{
		settings: settings,
		ticker:   time.Tick(time.Second / TPS),
	}

	evc.Connect("focus-out-event", s.Start)
	evc.Connect("focus-in-event", s.Stop)

	return &s
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
