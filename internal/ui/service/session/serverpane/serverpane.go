package serverpane

import (
	"github.com/gotk3/gotk3/gtk"
)

// Paned replaces gtk.Paned or gtk.Box to allow an optional second pane.
type Paned struct {
	gtk.IWidget
	Box *gtk.Box

	orien gtk.Orientation

	w1 gtk.IWidget
	w2 gtk.IWidget
}

// NewPaned creates a new empty pane.
func NewPaned(w1 gtk.IWidget, o gtk.Orientation) *Paned {
	// box holds either paned or w1.
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(w1, true, true, 0)

	return &Paned{
		IWidget: box,
		Box:     box,

		orien: o,
		w1:    w1,
	}
}

func (p *Paned) Destroy() {
	p.Box.Destroy()
	*p = Paned{}
}

func (p *Paned) Show() {
	p.Box.Show()
}

// AddSide adds a side widget. If a second widget is already added, then it is
// removed from the pane.
func (p *Paned) AddSide(w gtk.IWidget) {
	if p.w2 != nil {
		p.Box.Remove(p.w2)
	}

	p.w2 = w
	p.Box.PackStart(p.w2, true, true, 0)
	p.Box.SetChildPacking(p.w1, false, false, 0, gtk.PACK_START)
}

// Remove removes either w1 or w2. If neither matches, then nothing is done.
func (p *Paned) Remove(w gtk.IWidget) {
	switch w {
	case p.w1:
		panic("p.w1 must not be removed")
	case p.w2:
		p.Box.Remove(p.w2)
		p.w2 = nil
		p.Box.SetChildPacking(p.w1, true, true, 0, gtk.PACK_START)
	}
}
