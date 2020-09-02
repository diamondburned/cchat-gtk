package service

import (
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

type AppMenu struct {
	gtk.MenuButton
}

func NewAppMenu() *AppMenu {
	img, _ := gtk.ImageNew()
	img.SetFromPixbuf(icons.Logo256(24))
	img.Show()

	appmenu, _ := gtk.MenuButtonNew()
	appmenu.SetImage(img)
	appmenu.SetUsePopover(true)
	appmenu.SetHAlign(gtk.ALIGN_CENTER)
	appmenu.SetMarginStart(8)
	appmenu.SetMarginEnd(8)

	return &AppMenu{*appmenu}
}

func (a *AppMenu) SetSizeRequest(w, h int) {
	// Subtract the margin size.
	if w -= 8 * 2; w < 0 {
		w = 0
	}

	a.MenuButton.SetSizeRequest(w, h)
}

type Header struct {
	handy.HeaderBar

	MenuModel *glib.MenuModel

	AppMenu *AppMenu
	SvcName *gtk.Label
	SesMenu *actions.MenuButton
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

func NewHeader() *Header {
	menu := glib.MenuNew()
	menu.Append("Preferences", "app.preferences")
	menu.Append("Quit", "app.quit")

	appmenu := NewAppMenu()
	appmenu.Show()
	appmenu.SetMenuModel(&menu.MenuModel)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	sep.Show()
	primitives.AddClass(sep, "titlebutton")

	svcname, _ := gtk.LabelNew("cchat-gtk")
	svcname.SetXAlign(0)
	svcname.SetEllipsize(pango.ELLIPSIZE_END)
	svcname.Show()
	serviceNameCSS(svcname)

	sesmenu := actions.NewMenuButton()
	sesmenu.Show()
	sessionMenuCSS(sesmenu)

	header := handy.HeaderBarNew()
	header.SetProperty("spacing", 0)
	header.SetShowCloseButton(true)
	header.PackStart(appmenu)
	header.PackStart(sep)
	header.PackStart(svcname)
	header.PackEnd(sesmenu)

	// Hack to hide the title.
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	header.SetCustomTitle(b)

	return &Header{
		HeaderBar: *header,
		MenuModel: &menu.MenuModel,
		AppMenu:   appmenu,
		SvcName:   svcname,
		SesMenu:   sesmenu,
	}
}

func (h *Header) SetBreadcrumber(b traverse.Breadcrumber) {
	if b == nil {
		h.SvcName.SetText("cchat-gtk")
		return
	}

	if crumb := traverse.TryBreadcrumb(b); len(crumb) > 0 {
		h.SvcName.SetText(crumb[0])
	} else {
		h.SvcName.SetText("")
	}
}

func (h *Header) SetSessionMenu(s *session.Row) {
	h.SesMenu.Bind(s.ActionsMenu)
}

type sizeBinder interface {
	primitives.Connector
	GetAllocatedWidth() int
}

var _ sizeBinder = (*List)(nil)

func (h *Header) AppMenuBindSize(c sizeBinder) {
	c.Connect("size-allocate", func() {
		h.AppMenu.SetSizeRequest(c.GetAllocatedWidth(), -1)
	})
}
