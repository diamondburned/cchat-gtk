package ui

import (
	"html"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/gotk3/gotk3/gtk"
)

type header struct {
	*gtk.Box
	left  *headerLeft // middle-ish
	right *headerRight
}

func newHeader() *header {
	left := newHeaderLeft()
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

func (h *header) SetBreadcrumber(b breadcrumb.Breadcrumber) {
	if b == nil {
		h.right.breadcrumb.SetText("")
		return
	}

	var crumb = b.Breadcrumb()
	for i := range crumb {
		crumb[i] = html.EscapeString(crumb[i])
	}

	h.right.breadcrumb.SetMarkup(
		BreadcrumbSlash + " " + strings.Join(crumb, " "+BreadcrumbSlash+" "),
	)
}

func (h *header) SetSessionMenu(s *session.Row) {
	h.left.openmenu.Bind(s.ActionsMenu)
}

type headerLeft struct {
	*gtk.Box
	openmenu *actions.MenuButton
}

func newHeaderLeft() *headerLeft {
	openmenu := actions.NewMenuButton()
	openmenu.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(openmenu, false, false, 5)

	return &headerLeft{
		Box:      box,
		openmenu: openmenu,
	}
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
