package labeluri

import (
	"context"
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
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
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
	AvatarSize   = 96
	PopoverWidth = 250
	MaxWidth     = 350
	MaxHeight    = 350
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
	*BoundBox
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
	l.BoundBox = BindRichLabel(l)
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

type ReferenceHighlighter interface {
	HighlightReference(ref markup.ReferenceSegment)
}

// BoundBox is a box wrapping elements that can be interacted with from the
// parsed labels.
type BoundBox struct {
	label Labeler
	refer ReferenceHighlighter
}

func BindRichLabel(label Labeler) *BoundBox {
	bound := BoundBox{label: label}
	bind(label, bound.activate)
	return &bound
}

func (bound *BoundBox) activate(uri string, ptr gdk.Rectangle) bool {
	var output = bound.label.Output()

	switch segment := output.URISegment(uri).(type) {
	case markup.MentionSegment:
		popover := NewPopoverMentioner(bound.label, output.Input, segment)
		if popover != nil {
			popover.SetPointingTo(ptr)
			popover.Popup()
		}

		return true

	case markup.ReferenceSegment:
		if bound.refer != nil {
			bound.refer.HighlightReference(segment)
		}

		return true

	default:
		return false
	}
}

func (bound *BoundBox) SetReferenceHighlighter(refer ReferenceHighlighter) {
	bound.refer = refer
}

func PopoverMentioner(rel gtk.IWidget, input string, mention text.Segment) {
	if p := NewPopoverMentioner(rel, input, mention); p != nil {
		p.Popup()
	}
}

func NewPopoverMentioner(rel gtk.IWidget, input string, segment text.Segment) *gtk.Popover {
	var mention = segment.AsMentioner()
	if mention == nil {
		return nil
	}

	var info = mention.MentionInfo()
	if info.IsEmpty() {
		return nil
	}

	start, end := segment.Bounds()
	h := input[start:end]

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	// Do we have an image or an avatar?
	var url string
	var round bool

	if avatarer := segment.AsAvatarer(); avatarer != nil {
		url = avatarer.Avatar()
		round = true
	} else if imager := segment.AsImager(); imager != nil {
		url = imager.Image()
	}

	if url != "" {
		box.PackStart(popoverImg(url, round), false, false, 8)
	}

	head, _ := gtk.LabelNew(largeText(h))
	head.SetUseMarkup(true)
	head.SetLineWrap(true)
	head.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	head.SetMarginStart(8)
	head.SetMarginEnd(8)
	head.Show()
	box.PackStart(head, false, false, 0)

	// Left-align the label if we don't have an image.
	if url == "" {
		head.SetXAlign(0)
	}

	l, _ := gtk.LabelNew(markup.Render(info))
	l.SetUseMarkup(true)
	l.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	l.SetLineWrap(true)
	l.SetXAlign(0)
	l.SetMarginStart(8)
	l.SetMarginEnd(8)
	l.SetMarginTop(8)
	l.SetMarginBottom(8)
	l.Show()

	// Enable images???
	BindActivator(l)

	// Make a scrolling text.
	scr := scrollinput.NewVScroll(PopoverWidth)
	scr.Show()
	scr.Add(l)
	box.PackStart(scr, false, false, 0)

	p, _ := gtk.PopoverNew(rel)
	p.Add(box)
	p.SetSizeRequest(PopoverWidth, -1)
	return p
}

func largeText(text string) string {
	return fmt.Sprintf(
		`<span insert-hyphens="false" size="large">%s</span>`, html.EscapeString(text),
	)
}

// popoverImg creates a new button with an image for it, which is used for the
// avatar in the user popover.
func popoverImg(url string, round bool) gtk.IWidget {
	var btn *gtk.Button
	var img *gtk.Image
	var idl httputil.ImageContainer

	if round {
		b, _ := roundimage.NewButton()
		img = b.Image.GetImage()
		idl = b.Image
		btn = b.Button
	} else {
		img, _ = gtk.ImageNew()
		btn, _ = gtk.ButtonNew()
		btn.Add(img)
		idl = img
	}

	img.SetSizeRequest(AvatarSize, AvatarSize)
	img.SetHAlign(gtk.ALIGN_CENTER)
	img.Show()

	httputil.AsyncImage(context.Background(), idl, url)

	btn.SetHAlign(gtk.ALIGN_CENTER)
	btn.SetRelief(gtk.RELIEF_NONE)
	btn.Connect("clicked", func(*gtk.Button) { PromptOpen(url) })
	btn.Show()

	return btn
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
	connector.Connect("motion-notify-event", func(_ interface{}, ev *gdk.Event) {
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

			img.SetSizeRequest(w, h)
			img.SetFromIconName("image-loading", gtk.ICON_SIZE_BUTTON)
			img.Show()

			// Asynchronously fetch the image.
			httputil.AsyncImage(context.Background(), img, uri)

			btn, _ := gtk.ButtonNew()
			btn.Add(img)
			btn.SetRelief(gtk.RELIEF_NONE)
			btn.Connect("clicked", func(*gtk.Button) { PromptOpen(uri) })
			btn.Show()

			p, _ := gtk.PopoverNew(c)
			p.SetPointingTo(r)
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
