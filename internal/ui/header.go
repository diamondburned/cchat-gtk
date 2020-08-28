package ui

import (
	"html"
	"strings"

	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type header struct {
	left  *headerLeft // middle-ish
	right *headerRight
	menu  *glib.Menu
}

func newHeader() *header {
	menu := glib.MenuNew()
	menu.Append("Preferences", "app.preferences")
	menu.Append("Quit", "app.quit")

	left := newHeaderLeft()
	left.appmenu.SetMenuModel(&menu.MenuModel)

	right := newHeaderRight()

	group := handy.HeaderGroupNew()
	group.AddHeaderBar(&left.HeaderBar)
	group.AddHeaderBar(&right.HeaderBar)

	return &header{
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
	handy.HeaderBar

	appmenu *appMenu
	svcname *gtk.Label
	sesmenu *actions.MenuButton
}

var serviceNameCSS = primitives.PrepareClassCSS("service-name", `
	.service-name {
		margin-left: 14px;
	}
`)

var sessionMenuCSS = primitives.PrepareClassCSS("session-menu", `
	.session-menu {
		margin: 0 5px;
	}
`)

func newHeaderLeft() *headerLeft {
	appmenu := newAppMenu()
	appmenu.Show()

	// sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	// sep.Show()
	// primitives.AddClass(sep, "titlebutton")

	svcname, _ := gtk.LabelNew("cchat-gtk")
	svcname.SetXAlign(0)
	svcname.SetEllipsize(pango.ELLIPSIZE_END)
	svcname.Show()
	serviceNameCSS(svcname)

	sesmenu := actions.NewMenuButton()
	sesmenu.Show()
	sessionMenuCSS(sesmenu)

	header := handy.HeaderBarNew()
	header.SetShowCloseButton(true)
	header.PackStart(appmenu)
	// box.PackStart(sep, false, false, 0)
	header.PackStart(svcname)
	header.PackStart(sesmenu)

	return &headerLeft{
		HeaderBar: *header,
		appmenu:   appmenu,
		svcname:   svcname,
		sesmenu:   sesmenu,
	}
}

type headerRight struct {
	handy.HeaderBar

	breadcrumb *gtk.Label
}

var rightBreadcrumbCSS = primitives.PrepareClassCSS("right-breadcrumb", `
	.right-breadcrumb {
		margin: 0 14px;
	}
`)

func newHeaderRight() *headerRight {
	bc, _ := gtk.LabelNew(BreadcrumbSlash)
	bc.SetUseMarkup(true)
	bc.SetXAlign(0.0)
	bc.Show()
	rightBreadcrumbCSS(bc)

	header := handy.HeaderBarNew()
	header.SetShowCloseButton(true)
	header.PackStart(bc)
	header.Show()

	return &headerRight{
		HeaderBar:  *header,
		breadcrumb: bc,
	}
}
