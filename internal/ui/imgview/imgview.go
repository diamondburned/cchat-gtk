package imgview

import (
	"fmt"
	"html"
	"net/url"
	"path"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/c/labelutils"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/dialog"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
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

			// Make a new image that's asynchronously fetched inside a button.
			// This allows us to make it clickable.
			img, _ := gtk.ImageNewFromIconName("image-loading", gtk.ICON_SIZE_BUTTON)
			img.SetMarginStart(5)
			img.SetMarginEnd(5)
			img.SetMarginTop(5)
			img.SetMarginBottom(5)
			img.Show()

			// Cap the width and height if requested.
			var w, h = parser.FragmentImageSize(uri, MaxWidth, MaxHeight)
			httputil.AsyncImageSized(img, uri, w, h)

			btn, _ := gtk.ButtonNew()
			btn.Add(img)
			btn.SetRelief(gtk.RELIEF_NONE)
			btn.Connect("clicked", func() { PromptOpen(uri) })
			btn.Show()

			p, _ := gtk.PopoverNew(c)
			p.SetPointingTo(r)
			p.Connect("closed", img.Destroy) // on close, destroy image
			p.Add(btn)
			p.Popup()

		default:
			PromptOpen(uri)
		}

		// Never let Gtk open the dialog.
		return true
	})
}

const urlPrompt = `This link leads to the following URL:
<span weight="bold"><a href="%[1]s">%[1]s</a></span>
Click <b>Open</b> to proceed.`

var warnLabelCSS = primitives.PrepareCSS(`
	label {
		padding: 4px 8px;
	}
`)

// PromptOpen shows a dialog asking if the URL should be opened.
func PromptOpen(uri string) {
	// Format the prompt body.
	l, _ := gtk.LabelNew("")
	l.SetJustify(gtk.JUSTIFY_CENTER)
	l.SetLineWrap(true)
	l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	l.Show()
	l.SetMarkup(fmt.Sprintf(urlPrompt, html.EscapeString(uri)))

	// Style the label.
	primitives.AttachCSS(l, warnLabelCSS)

	// Disable hyphens on line wraps.
	labelutils.AddAttr(l, labelutils.InsertHyphens(false))

	open := func() {
		if err := open.Start(uri); err != nil {
			log.Error(errors.Wrap(err, "Failed to open URL after confirm"))
		}
	}

	// Prompt the user if they want to open the URL.
	dlg := dialog.NewModal(l, "Caution", "Open", open)
	dlg.SetSizeRequest(350, 100)

	// Add a class to the dialog to allow theming.
	primitives.AddClass(dlg, "url-warning")

	// On link click, close the dialog.
	l.Connect("activate-link", func(l *gtk.Label, uri string) bool {
		// Close the dialog.
		dlg.Destroy()
		// Open the link anyway.
		open()
		// Return true since we handled the event.
		return true
	})

	// Show the dialog.
	dlg.Show()
}

// ext parses and sanitizes the extension to something comparable.
func ext(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return strings.ToLower(path.Ext(uri))
	}

	return strings.ToLower(path.Ext(u.Path))
}
