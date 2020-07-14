package completion

import (
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

const (
	ImageSmall   = 25
	ImageLarge   = 40
	ImagePadding = 6
)

var ppIcon = []imgutil.Processor{imgutil.Round(true)}

type View struct {
	*completion.Completer
	entries   []cchat.CompletionEntry
	completer cchat.ServerMessageSendCompleter
}

func New(text *gtk.TextView) *View {
	v := &View{}
	c := completion.NewCompleter(text, v)
	v.Completer = c

	return v
}

func (v *View) Reset() {
	v.SetCompleter(nil)
}

func (v *View) SetCompleter(completer cchat.ServerMessageSendCompleter) {
	v.Clear()
	v.Hide()
	v.completer = completer
}

func (v *View) Update(words []string, i int) []gtk.IWidget {
	// If we don't have a completer, then don't run.
	if v.completer == nil {
		return nil
	}

	v.entries = v.completer.CompleteMessage(words, i)

	var widgets = make([]gtk.IWidget, len(v.entries))

	for i, entry := range v.entries {
		// Container that holds the label.
		lbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
		lbox.SetVAlign(gtk.ALIGN_CENTER)
		lbox.Show()

		// Label for the primary text.
		l := rich.NewLabel(entry.Text)
		l.Show()
		lbox.PackStart(l, false, false, 0)

		// Get the iamge size so we can change and use if needed. The default
		var size = ImageSmall
		if !entry.Secondary.Empty() {
			size = ImageLarge

			s := rich.NewLabel(text.Rich{})
			s.SetMarkup(fmt.Sprintf(
				`<span alpha="50%%" size="small">%s</span>`,
				markup.Render(entry.Secondary),
			))
			s.Show()

			lbox.PackStart(s, false, false, 0)
		}

		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		b.PackEnd(lbox, true, true, ImagePadding)
		b.Show()

		// Do we have an icon?
		if entry.IconURL != "" {
			img, _ := gtk.ImageNew()
			img.SetMarginStart(ImagePadding)
			img.SetSizeRequest(size, size)
			img.Show()

			// Prepend the image into the box.
			b.PackEnd(img, false, false, 0)

			var pps []imgutil.Processor
			if !entry.Image {
				pps = ppIcon
			}

			httputil.AsyncImageSized(img, entry.IconURL, size, size, pps...)
		}

		widgets[i] = b
	}

	return widgets
}

func (v *View) Word(i int) string {
	return v.entries[i].Raw
}
