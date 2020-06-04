package autoscroll

import "github.com/gotk3/gotk3/gtk"

type ScrolledWindow struct {
	gtk.ScrolledWindow
	vadj     gtk.Adjustment
	bottomed bool // :floshed:
}

func NewScrolledWindow() *ScrolledWindow {
	gtksw, _ := gtk.ScrolledWindowNew(nil, nil)
	gtksw.SetProperty("propagate-natural-height", true)

	sw := &ScrolledWindow{*gtksw, *gtksw.GetVAdjustment(), true} // bottomed by default
	sw.Connect("size-allocate", func(_ *gtk.ScrolledWindow) {
		// We can't really trust Gtk to be competent.
		if sw.bottomed {
			sw.ScrollToBottom()
		}
	})
	sw.vadj.Connect("value-changed", func(adj *gtk.Adjustment) {
		// Manually check if we're anchored on scroll.
		sw.bottomed = (adj.GetUpper() - adj.GetPageSize()) <= adj.GetValue()
	})

	return sw
}

func (s *ScrolledWindow) Bottomed() bool {
	return s.bottomed
}

// GetVAdjustment overrides gtk.ScrolledWindow's.
func (s *ScrolledWindow) GetVAdjustment() *gtk.Adjustment {
	return &s.vadj
}

func (s *ScrolledWindow) ScrollToBottom() {
	s.vadj.SetValue(s.vadj.GetUpper())
}
