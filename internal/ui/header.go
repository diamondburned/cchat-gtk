package ui

import (
	"html"
	"strings"

	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type header struct {
	*gtk.Box
	left  *headerLeft // middle-ish
	right *headerRight
	menu  *glib.Menu
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

	menu := glib.MenuNew()
	menu.Append("Preferences", "app.preferences")
	menu.Append("Quit", "app.quit")

	left.appmenu.SetMenuModel(&menu.MenuModel)

	// TODO
	return &header{
		box,
		left,
		right,
		menu,
	}
}

// const BreadcrumbSlash = `<span rise="-1024" size="x-large">❭</span>`
const BreadcrumbSlash = " 〉"

func (h *header) SetBreadcrumber(b traverse.Breadcrumber) {
	if b == nil {
		h.right.breadcrumb.SetText("")
		return
	}

	var crumb = b.Breadcrumb()

	if len(crumb) > 0 {
		h.left.svcname.SetText(crumb[0])
	} else {
		h.left.svcname.SetText("")
	}

	for i := range crumb {
		crumb[i] = html.EscapeString(crumb[i])
	}

	h.right.breadcrumb.SetMarkup(
		BreadcrumbSlash + " " + strings.Join(crumb, " "+BreadcrumbSlash+" "),
	)
}

func (h *header) SetSessionMenu(s *session.Row) {
	h.left.sesmenu.Bind(s.ActionsMenu)
}

type appMenu struct {
	*gtk.MenuButton
}

func newAppMenu() *appMenu {
	img, _ := gtk.ImageNew()
	img.SetFromPixbuf(icons.Logo256(24))
	img.Show()

	appmenu, _ := gtk.MenuButtonNew()
	appmenu.SetImage(img)
	appmenu.SetUsePopover(true)
	appmenu.SetHAlign(gtk.ALIGN_CENTER)
	appmenu.SetMarginStart(8)
	appmenu.SetMarginEnd(8)

	return &appMenu{appmenu}
}

func (a *appMenu) SetSizeRequest(w, h int) {
	// Subtract the margin size.
	if w -= 8 * 2; w < 0 {
		w = 0
	}

	a.MenuButton.SetSizeRequest(w, h)
}

type headerLeft struct {
	*gtk.Box
	appmenu *appMenu
	svcname *gtk.Label
	sesmenu *actions.MenuButton
}

func newHeaderLeft() *headerLeft {
	appmenu := newAppMenu()
	appmenu.Show()

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	sep.Show()
	primitives.AddClass(sep, "titlebutton")

	svcname, _ := gtk.LabelNew("")
	svcname.SetXAlign(0)
	svcname.SetEllipsize(pango.ELLIPSIZE_END)
	svcname.Show()
	svcname.SetMarginStart(14)

	sesmenu := actions.NewMenuButton()
	sesmenu.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(appmenu, false, false, 0)
	box.PackStart(sep, false, false, 0)
	box.PackStart(svcname, true, true, 0)
	box.PackStart(sesmenu, false, false, 5)

	return &headerLeft{
		Box:     box,
		appmenu: appmenu,
		svcname: svcname,
		sesmenu: sesmenu,
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
