package buttonoverlay

import "github.com/gotk3/gotk3/gtk"

type Widget interface {
	gtk.IWidget
	SetMarginEnd(int)
	SetSizeRequest(int, int)
	SetHAlign(gtk.Align)
}

var _ Widget = (*gtk.Widget)(nil)

type Button interface {
	Widget
	// Bin
	GetChild() (gtk.IWidget, error)
	// Container
	Add(gtk.IWidget)
	Remove(gtk.IWidget)
	// Button
	SetRelief(gtk.ReliefStyle)
}

var _ Button = (*gtk.Button)(nil)

// Wrap wraps maincontent inside an overlay with smallbutton placed rightmost on
// top of the content. It will also set the margins and aligns widgets.
func Wrap(maincontent Widget, smallbutton Button, size int) *gtk.Overlay {
	maincontent.SetMarginEnd(size)
	smallbutton.SetSizeRequest(size, size)
	smallbutton.SetHAlign(gtk.ALIGN_END)
	smallbutton.SetRelief(gtk.RELIEF_NONE)

	o, _ := gtk.OverlayNew()
	o.Add(maincontent)
	o.AddOverlay(smallbutton)
	o.Show()

	return o
}

// Take takes over the given button and replaces its content with the wrapped
// overlay, which has the old content as well as the smaller button on top.
func Take(b, smallbutton Button, size int) {
	childv, _ := b.GetChild()
	widget := childv.ToWidget()

	// This will unreference.
	b.Remove(widget)
	// Wrap will reference.
	b.Add(Wrap(widget, smallbutton, size))
}
