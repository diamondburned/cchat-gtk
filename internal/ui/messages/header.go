package messages

import (
	"html"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// const BreadcrumbSlash = `<span rise="-1024" size="x-large">❭</span>`
const BreadcrumbSlash = " 〉"

type Header struct {
	handy.HeaderBar

	ShowBackBtn *gtk.Revealer
	BackButton  *gtk.Button
	Breadcrumb  *gtk.Label
	ShowMembers *gtk.ToggleButton

	breadcrumbs []string
	minicrumbs  bool
}

var backButtonCSS = primitives.PrepareClassCSS("back-button", `
	.back-button {
		margin-left: 14px;
	}
`)

var rightBreadcrumbCSS = primitives.PrepareClassCSS("right-breadcrumb", `
	.right-breadcrumb {
		margin-left: 14px;
	}
`)

func NewHeader() *Header {
	bk, _ := gtk.ButtonNewFromIconName("go-previous-symbolic", gtk.ICON_SIZE_BUTTON)
	bk.SetVAlign(gtk.ALIGN_CENTER)
	bk.Show()
	backButtonCSS(bk)

	rbk, _ := gtk.RevealerNew()
	rbk.Add(bk)
	rbk.SetRevealChild(false)
	rbk.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_LEFT)
	rbk.SetTransitionDuration(50)
	rbk.Show()

	bc, _ := gtk.LabelNew(BreadcrumbSlash)
	bc.SetUseMarkup(true)
	bc.SetXAlign(0.0)
	bc.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
	bc.SetSingleLineMode(true)
	bc.SetHExpand(true)
	bc.SetMaxWidthChars(75)
	bc.Show()
	rightBreadcrumbCSS(bc)

	memberIcon, _ := gtk.ImageNewFromIconName("system-users-symbolic", gtk.ICON_SIZE_BUTTON)
	memberIcon.Show()

	mb, _ := gtk.ToggleButtonNew()
	mb.SetVAlign(gtk.ALIGN_CENTER)
	mb.SetImage(memberIcon)
	mb.SetActive(false)
	mb.SetSensitive(false)

	header := handy.HeaderBarNew()
	header.SetShowCloseButton(true)
	header.PackStart(rbk)
	header.PackStart(bc)
	header.PackEnd(mb)
	header.Show()

	// Hack to hide the title.
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	header.SetCustomTitle(b)

	return &Header{
		HeaderBar:   *header,
		ShowBackBtn: rbk,
		BackButton:  bk,
		Breadcrumb:  bc,
		ShowMembers: mb,
	}
}

func (h *Header) Reset() {
	h.SetBreadcrumber(nil)
}

func (h *Header) OnBackPressed(fn func()) {
	h.BackButton.Connect("clicked", fn)
}

func (h *Header) OnShowMembersToggle(fn func(show bool)) {
	h.ShowMembers.Connect("toggled", func() {
		fn(h.ShowMembers.GetActive())
	})
}

func (h *Header) SetShowBackButton(show bool) {
	h.ShowBackBtn.SetRevealChild(show)
}

func (h *Header) SetCanShowMembers(canShow bool) {
	if canShow {
		h.ShowMembers.Show()
		h.ShowMembers.SetSensitive(true)
	} else {
		h.ShowMembers.Hide()
		h.ShowMembers.SetSensitive(false)
	}
}

// SetMiniBreadcrumb sets whether or not the breadcrumb should display the full
// label.
func (h *Header) SetMiniBreadcrumb(mini bool) {
	h.minicrumbs = mini
	h.updateBreadcrumb()
}

// updateBreadcrumb updates the breadcrumb label from the local state.
func (h *Header) updateBreadcrumb() {
	switch {
	case len(h.breadcrumbs) == 0:
		h.Breadcrumb.SetText("")

	case h.minicrumbs:
		h.Breadcrumb.SetMarkup(h.breadcrumbs[len(h.breadcrumbs)-1])

	default:
		h.Breadcrumb.SetMarkup(
			BreadcrumbSlash + " " + strings.Join(h.breadcrumbs, " "+BreadcrumbSlash+" "),
		)
	}
}

func (h *Header) SetBreadcrumber(b traverse.Breadcrumber) {
	if b == nil {
		h.breadcrumbs = nil
		h.updateBreadcrumb()
		return
	}

	h.breadcrumbs = traverse.TryBreadcrumb(b)
	if len(h.breadcrumbs) < 2 {
		return
	}

	// Skip the service name and username.
	h.breadcrumbs = h.breadcrumbs[2:]

	for i := range h.breadcrumbs {
		h.breadcrumbs[i] = html.EscapeString(h.breadcrumbs[i])
	}

	h.updateBreadcrumb()
}
