package imgview

import (
	"net/url"
	"path"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const (
	MaxWidth  = 350
	MaxHeight = 350
)

type WidgetConnector interface {
	gtk.IWidget
	primitives.Connector
}

var _ WidgetConnector = (*gtk.Label)(nil)

func BindTooltip(connector WidgetConnector) {
	// This implementation doesn't seem like a good idea. First off, is the
	// closure really garbage collected? If it's not, then we have some huge
	// issues. Second, if the closure is garbage collected, then when? If it's
	// not garbage collecteed, then not only are we leaking 2 float64s per
	// message, but we're also keeping alive the widget.

	var x, y float64
	connector.Connect("motion-notify-event", func(w gtk.IWidget, ev *gdk.Event) {
		x, y = gdk.EventMotionNewFromEvent(ev).MotionVal()
	})

	connector.Connect("activate-link", func(c WidgetConnector, uri string) bool {
		switch ext(uri) {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			// Make a new rectangle to use in the popover.
			r := gdk.Rectangle{}
			r.SetX(int(x))
			r.SetY(int(y))

			// Make a new image that's asynchronously fetched.
			img, _ := gtk.ImageNewFromIconName("image-loading", gtk.ICON_SIZE_BUTTON)
			img.SetMarginStart(5)
			img.SetMarginEnd(5)
			img.SetMarginTop(5)
			img.SetMarginBottom(5)
			img.Show()

			// Cap the width and height if requested.
			var w, h = parser.FragmentImageSize(uri, MaxWidth, MaxHeight)
			httputil.AsyncImageSized(img, uri, w, h)

			p, _ := gtk.PopoverNew(c)
			p.SetPointingTo(r)
			p.Connect("closed", img.Destroy) // on close, destroy image
			p.Add(img)
			p.Popup()

			return true

		default:
			return false
		}
	})
}

// ext parses and sanitizes the extension to something comparable.
func ext(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return strings.ToLower(path.Ext(uri))
	}

	return strings.ToLower(path.Ext(u.Path))
}
