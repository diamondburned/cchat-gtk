package ui

import (
	"html"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/gotk3/gotk3/gtk"
)

type header struct {
	*gtk.Box
	left  *gtk.Box // TODO
	right *headerRight
}

func newHeader() *header {
	left, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	left.SetSizeRequest(leftMinWidth, -1)
	left.Show()

	right := newHeaderRight()
	right.Show()

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	separator.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(left, false, false, 0)
	box.PackStart(separator, false, false, 0)
	box.PackStart(right, true, true, 0)
	box.Show()

	// TODO
	return &header{
		box,
		left,
		right,
	}
}

const BreadcrumbSlash = `<span weight="light" rise="-1024" size="x-large">/</span>`

func (h *header) SetBreadcrumb(b breadcrumb.Breadcrumb) {
	for i := range b {
		b[i] = html.EscapeString(b[i])
	}

	h.right.breadcrumb.SetMarkup(
		BreadcrumbSlash + " " + strings.Join(b, " "+BreadcrumbSlash+" "),
	)
}

type headerRight struct {
	*gtk.Box
	breadcrumb *gtk.Label
}

func newHeaderRight() *headerRight {
	bc, _ := gtk.LabelNew(BreadcrumbSlash)
	bc.SetUseMarkup(true)
	bc.SetXAlign(0.0)
	bc.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(bc, true, true, 14)
	box.Show()

	return &headerRight{
		Box:        box,
		breadcrumb: bc,
	}
}
