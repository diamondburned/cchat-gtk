package autoscroll

import "github.com/gotk3/gotk3/gtk"

type ScrolledWindow struct {
	gtk.ScrolledWindow
	vadj     gtk.Adjustment
	Bottomed bool // :floshed:
}

func NewScrolledWindow() *ScrolledWindow {
	gtksw, _ := gtk.ScrolledWindowNew(nil, nil)
	gtksw.SetProperty("propagate-natural-height", true)
	gtksw.SetProperty("window-placement", gtk.CORNER_BOTTOM_LEFT)

	sw := &ScrolledWindow{*gtksw, *gtksw.GetVAdjustment(), true} // bottomed by default
	sw.Connect("size-allocate", func(_ *gtk.ScrolledWindow) {
		// We can't really trust Gtk to be competent.
		if sw.Bottomed {
			sw.ScrollToBottom()
		}
	})
	sw.vadj.Connect("value-changed", func(adj *gtk.Adjustment) {
		// Manually check if we're anchored on scroll.
		sw.Bottomed = (adj.GetUpper() - adj.GetPageSize()) <= adj.GetValue()
	})

	return sw
}

// GetVAdjustment overrides gtk.ScrolledWindow's.
func (s *ScrolledWindow) GetVAdjustment() *gtk.Adjustment {
	return &s.vadj
}

func (s *ScrolledWindow) ScrollToBottom() {
	s.vadj.SetValue(s.vadj.GetUpper())
}
