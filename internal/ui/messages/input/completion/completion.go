package completion

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/utils/split"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

const (
	ImageSize    = 20
	ImagePadding = 10
)

// var completionQueue chan func()

// func init() {
// 	completionQueue = make(chan func(), 1)
// 	go func() {
// 		for fn := range completionQueue {
// 			fn()
// 		}
// 	}()
// }

type View struct {
	*gtk.Revealer
	Scroll *gtk.ScrolledWindow

	List    *gtk.ListBox
	entries []cchat.CompletionEntry

	text   *gtk.TextView
	buffer *gtk.TextBuffer

	// state
	completer cchat.ServerMessageSendCompleter
	offset    int
}

func New(text *gtk.TextView) *View {
	list, _ := gtk.ListBoxNew()
	list.SetSelectionMode(gtk.SELECTION_BROWSE)
	list.Show()

	primitives.AddClass(list, "completer")

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.Add(list)
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.SetProperty("propagate-natural-height", true)
	scroll.SetProperty("max-content-height", 250)
	scroll.Show()

	// Bind scroll adjustments.
	list.SetFocusHAdjustment(scroll.GetHAdjustment())
	list.SetFocusVAdjustment(scroll.GetVAdjustment())

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.SetTransitionDuration(50)
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_UP)
	rev.Add(scroll)
	rev.Show()

	buffer, _ := text.GetBuffer()

	v := &View{
		Revealer: rev,
		Scroll:   scroll,
		List:     list,
		text:     text,
		buffer:   buffer,
	}

	text.Connect("key-press-event", completion.KeyDownHandler(list, text.GrabFocus))
	buffer.Connect("changed", func() {
		// Clear the list first.
		v.Clear()
		// Re-run the list.
		v.Run()
	})

	list.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		completion.SwapWord(v.buffer, v.entries[r.GetIndex()].Raw, v.offset)
		v.Clear()
		v.text.GrabFocus() // TODO: remove, maybe not needed
	})

	return v
}

// SetMarginStart sets the left margin but account for images as well.
func (v *View) SetMarginStart(pad int) {
	pad = pad - (ImagePadding*2 + ImageSize - 2) // subtracting 2 for no reason
	if pad < 0 {
		pad = 0
	}
	v.Revealer.SetMarginStart(pad)
}

func (v *View) Reset() {
	v.SetCompleter(nil)
}

func (v *View) SetCompleter(completer cchat.ServerMessageSendCompleter) {
	v.Clear()
	v.completer = completer
}

func (v *View) Clear() {
	// Do we have anything in the slice? If not, then we don't need to run
	// again. We do need to keep RevealChild consistent with this, however.
	if v.entries == nil {
		return
	}

	// Since we don't store the widgets inside the list, we'll manually iterate
	// and remove.
	v.List.GetChildren().Foreach(func(i interface{}) {
		w := i.(gtk.IWidget).ToWidget()
		v.List.Remove(w)
		w.Destroy()
	})

	// Set entries to nil to free up the slice.
	v.entries = nil
	// Set offset to 0 to reset.
	v.offset = 0

	// Hide the list.
	v.SetRevealChild(false)
}

func (v *View) Run() {
	// If we don't have a completer, then don't run.
	if v.completer == nil {
		return
	}

	text, offset := v.getInputState()
	words, index := split.SpaceIndexed(text, offset)

	// If the input is empty.
	if len(words) == 0 {
		return
	}

	v.offset = offset
	v.entries = v.completer.CompleteMessage(words, index)

	if len(v.entries) == 0 {
		return
	}

	// Reveal if needed be.
	v.SetRevealChild(true)

	// TODO: make entries reuse pixbuf.

	for i, entry := range v.entries {
		l := rich.NewLabel(entry.Text)
		l.Show()

		img, _ := gtk.ImageNew()
		img.SetSizeRequest(ImageSize, ImageSize)
		img.Show()

		// Do we have an icon?
		if entry.IconURL != "" {
			httputil.AsyncImageSized(img, entry.IconURL, ImageSize, ImageSize, imgutil.Round(true))
		}

		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		b.PackStart(img, false, false, ImagePadding)
		b.PackStart(l, true, true, 0)
		b.Show()

		r, _ := gtk.ListBoxRowNew()
		r.Add(b)
		r.Show()

		v.List.Add(r)

		// Select the first item.
		if i == 0 {
			v.List.SelectRow(r)
		}
	}
}

func (v *View) getInputState() (string, int) {
	return completion.State(v.buffer)
}
