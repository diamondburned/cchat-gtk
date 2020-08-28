package messages

import (
	"html"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

// const BreadcrumbSlash = `<span rise="-1024" size="x-large">❭</span>`
const BreadcrumbSlash = " 〉"

type Header struct {
	handy.HeaderBar

	Breadcrumb *gtk.Label
}

var rightBreadcrumbCSS = primitives.PrepareClassCSS("right-breadcrumb", `
	.right-breadcrumb {
		margin: 0 14px;
	}
`)

func NewHeader() *Header {
	bc, _ := gtk.LabelNew(BreadcrumbSlash)
	bc.SetUseMarkup(true)
	bc.SetXAlign(0.0)
	bc.Show()
	rightBreadcrumbCSS(bc)

	header := handy.HeaderBarNew()
	header.SetShowCloseButton(true)
	header.PackStart(bc)
	header.Show()

	return &Header{
		HeaderBar:  *header,
		Breadcrumb: bc,
	}
}

func (h *Header) Reset() {
	h.SetBreadcrumber(nil)
}

func (h *Header) SetBreadcrumber(b traverse.Breadcrumber) {
	if b == nil {
		h.Breadcrumb.SetText("")
		return
	}

	var crumb = b.Breadcrumb()
	for i := range crumb {
		crumb[i] = html.EscapeString(crumb[i])
	}

	h.Breadcrumb.SetMarkup(
		BreadcrumbSlash + " " + strings.Join(crumb, " "+BreadcrumbSlash+" "),
	)
}
