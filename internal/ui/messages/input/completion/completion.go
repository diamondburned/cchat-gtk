package completion

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

const (
	ImageSize    = 25
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
		l := rich.NewLabel(entry.Text)
		l.Show()

		img, _ := gtk.ImageNew()

		// Do we have an icon?
		if entry.IconURL != "" {
			img.SetMarginStart(ImagePadding)
			img.SetSizeRequest(ImageSize, ImageSize)
			img.Show()

			var pps []imgutil.Processor
			if !entry.Image {
				pps = ppIcon
			}

			httputil.AsyncImageSized(img, entry.IconURL, ImageSize, ImageSize, pps...)
		}

		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		b.PackStart(img, false, false, 0) // image has pad left
		b.PackStart(l, true, true, ImagePadding)
		b.Show()

		widgets[i] = b
	}

	return widgets
}

func (v *View) Word(i int) string {
	return v.entries[i].Raw
}
