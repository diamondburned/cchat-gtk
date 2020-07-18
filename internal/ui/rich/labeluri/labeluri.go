package labeluri

import (
	"fmt"
	"html"
	"net/url"
	"path"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/dialog"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
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

// Labeler implements a rich label that stores an output state.
type Labeler interface {
	WidgetConnector
	rich.Labeler
	Output() markup.RenderOutput
}

// Label implements a label that's already bounded to the markup URI handlers.
type Label struct {
	*rich.Label
	output markup.RenderOutput
}

var (
	_ Labeler           = (*Label)(nil)
	_ rich.SuperLabeler = (*Label)(nil)
)

func NewLabel(txt text.Rich) *Label {
	l := &Label{}
	l.Label = rich.NewInheritLabel(l)
	l.Label.SetLabelUnsafe(txt) // test

	// Bind and return.
	BindRichLabel(l)
	return l
}

func (l *Label) Reset() {
	l.output = markup.RenderOutput{}
}

func (l *Label) SetLabelUnsafe(content text.Rich) {
	l.output = markup.RenderCmplx(content)
	l.SetMarkup(l.output.Markup)
}

// Output returns the label's markup output. This function is NOT
// thread-safe.
func (l *Label) Output() markup.RenderOutput {
	return l.output
}

// SetOutput sets the internal output and label.
func (l *Label) SetOutput(o markup.RenderOutput) {
	l.output = o
	l.SetMarkup(o.Markup)
}

func BindRichLabel(label Labeler) {
	bind(label, func(uri string, ptr gdk.Rectangle) bool {
		var output = label.Output()

		if mention := output.IsMention(uri); mention != nil {
			if p := popoverMentioner(label, mention); p != nil {
				p.SetPointingTo(ptr)
				p.Popup()
			}

			return true
		}

		return false
	})
}

func PopoverMentioner(rel gtk.IWidget, mention text.Mentioner) {
	if p := popoverMentioner(rel, mention); p != nil {
		p.Popup()
	}
}

func popoverMentioner(rel gtk.IWidget, mention text.Mentioner) *gtk.Popover {
	var info = mention.MentionInfo()
	if info.Empty() {
		return nil
	}

	l, _ := gtk.LabelNew(markup.Render(info))
	l.SetUseMarkup(true)
	l.SetXAlign(0)
	l.Show()

	// Enable images???
	BindActivator(l)

	p, _ := gtk.PopoverNew(rel)
	p.Add(l)
	p.Connect("destroy", l.Destroy)
	return p
}

func BindActivator(connector WidgetConnector) {
	bind(connector, nil)
}

// bind connects activate-link. If activator returns true, then nothing is done.
// Activator can be nil.
func bind(connector WidgetConnector, activator func(uri string, r gdk.Rectangle) bool) {
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
		// Make a new rectangle to use in the popover.
		r := gdk.Rectangle{}
		r.SetX(int(x))
		r.SetY(int(y))

		if activator != nil && activator(uri, r) {
			return true
		}

		switch ext(uri) {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			// Make a new image that's asynchronously fetched inside a button.
			// Cap the width and height if requested.
			var w, h, round = markup.FragmentImageSize(uri, MaxWidth, MaxHeight)

			var img *gtk.Image
			if !round {
				img, _ = gtk.ImageNew()
			} else {
				r, _ := roundimage.NewImage(0)
				img = r.Image
			}

			img.SetFromIconName("image-loading", gtk.ICON_SIZE_BUTTON)
			img.Show()

			// Asynchronously fetch the image.
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

			return true
		}

		PromptOpen(uri)

		// Never let Gtk open the dialog.
		return true
	})
}

const urlPrompt = `This link leads to the following URL:
<span weight="bold" insert_hyphens="false"><a href="%[1]s">%[1]s</a></span>
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

	open := func(m *dialog.Modal) {
		// Close the dialog.
		m.Destroy()
		// Open the link.
		if err := open.Start(uri); err != nil {
			log.Error(errors.Wrap(err, "Failed to open URL after confirm"))
		}
	}

	// Prompt the user if they want to open the URL.
	dlg := dialog.NewModal(l, "Caution", "_Open", open)
	dlg.SetSizeRequest(350, 100)

	// Style the button to have a color.
	primitives.SuggestAction(dlg.Action)

	// Add a class to the dialog to allow theming.
	primitives.AddClass(dlg, "url-warning")

	// On link click, close the dialog, open the link ourselves, then return.
	l.Connect("activate-link", func(l *gtk.Label, uri string) bool {
		open(dlg)
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
