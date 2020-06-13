package completion

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/utils/split"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
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

	text.Connect("key-press-event", v.inputKeyDown)
	buffer.Connect("changed", func() {
		// Clear the list first.
		v.Clear()
		// Re-run the list.
		v.Run()
	})

	list.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		// Get iter for word replacing.
		start, end := getWordIters(v.buffer, v.offset)

		// Get the selected word.
		i := r.GetIndex()
		entry := v.entries[i]

		// Replace the word.
		v.buffer.Delete(start, end)
		v.buffer.Insert(start, entry.Raw+" ")

		// Clear the list.
		v.Clear()

		// Reset the focus.
		v.text.GrabFocus()
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
	// obtain current state
	mark := v.buffer.GetInsert()
	iter := v.buffer.GetIterAtMark(mark)

	// obtain the input string and the current cursor position
	start, end := v.buffer.GetBounds()
	text, _ := v.buffer.GetText(start, end, true)
	offset := iter.GetOffset()

	return text, offset
}

// inputKeyDown handles keypresses such as Enter and movements.
func (v *View) inputKeyDown(_ *gtk.TextView, ev *gdk.Event) (stop bool) {
	// Do we have any entries? If not, don't bother.
	if len(v.entries) == 0 {
		// passthrough.
		return false
	}

	var evKey = gdk.EventKeyNewFromEvent(ev)
	var key = evKey.KeyVal()

	switch key {
	// Did we press an arrow key?
	case gdk.KEY_Up, gdk.KEY_Down:
		// Yes, start moving the list up and down.
		i := v.List.GetSelectedRow().GetIndex()

		switch key {
		case gdk.KEY_Up:
			if i--; i < 0 {
				i = len(v.entries) - 1
			}
		case gdk.KEY_Down:
			if i++; i >= len(v.entries) {
				i = 0
			}
		}

		row := v.List.GetRowAtIndex(i)
		row.GrabFocus()       // scroll
		v.List.SelectRow(row) // select
		v.text.GrabFocus()    // unfocus

	// Did we press the Enter or Tab key?
	case gdk.KEY_Return, gdk.KEY_Tab:
		// Activate the current row.
		row := v.List.GetSelectedRow()
		row.Activate()

	default:
		// don't passthrough events if none matches.
		return false
	}

	return true
}

func getWordIters(buf *gtk.TextBuffer, offset int) (start, end *gtk.TextIter) {
	iter := buf.GetIterAtOffset(offset)

	var ok bool

	// Seek backwards for space or start-of-line:
	_, start, ok = iter.BackwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
	if !ok {
		start = buf.GetStartIter()
	}

	// Seek forwards for space or end-of-line:
	_, end, ok = iter.ForwardSearch(" ", gtk.TEXT_SEARCH_TEXT_ONLY, nil)
	if !ok {
		end = buf.GetEndIter()
	}

	return
}
