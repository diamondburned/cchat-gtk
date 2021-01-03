package completion

import (
	"context"
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/cchat/utils/split"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

const (
	ImageSmall   = 25
	ImageLarge   = 40
	ImagePadding = 6
)

// post-processor icon
var ppIcon = []imgutil.Processor{imgutil.Round(true)}

type Completer struct {
	Input   *gtk.TextView
	Buffer  *gtk.TextBuffer
	List    *gtk.ListBox
	Popover *gtk.Popover
	popdown bool

	Splitter split.SplitFunc

	words  []string
	index  int64
	cursor int64

	entries   []cchat.CompletionEntry
	completer cchat.Completer
}

func WrapCompleter(input *gtk.TextView) {
	NewCompleter(input)
}

func NewCompleter(input *gtk.TextView) *Completer {
	l, _ := gtk.ListBoxNew()
	l.Show()

	s := scrollinput.NewVScroll(150)
	s.Add(l)
	s.Show()

	p := NewPopover(input)
	p.Add(s)

	input.Connect("key-press-event", KeyDownHandler(l, input.GrabFocus))
	ibuf, _ := input.GetBuffer()

	c := &Completer{
		Input:    input,
		Buffer:   ibuf,
		List:     l,
		Popover:  p,
		Splitter: split.SpaceIndexed,
	}

	// This one is for buffer modification.
	ibuf.Connect("end-user-action", func(interface{}) { c.onChange() })
	// This one is for when the cursor moves.
	input.Connect("move-cursor", func(interface{}) { c.onChange() })

	l.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		SwapWord(ibuf, c.entries[r.GetIndex()].Raw, c.cursor)
		c.onChange() // signal change
		c.Popdown()
		input.GrabFocus()
	})

	return c
}

// SetCompleter sets the current completer. If completer is nil, then the
// completer is disabled.
func (c *Completer) SetCompleter(completer cchat.Completer) {
	c.Popdown()
	c.completer = completer
}

func (c *Completer) Reset() {
	c.SetCompleter(nil)
}

func (c *Completer) Popup() {
	if c.popdown {
		c.Popover.Popup()
		c.popdown = false
	}
}

func (c *Completer) Popdown() {
	if !c.popdown {
		c.Popover.Popdown()
		c.popdown = true
		c.Clear()
	}
}

func (c *Completer) Clear() {
	primitives.RemoveChildren(c.List)
}

// Words returns the buffer content split into words.
func (c *Completer) Content() []string {
	// This method not to be confused with c.words, which contains the state of
	// completer words.

	text, _ := c.Buffer.GetText(c.Buffer.GetStartIter(), c.Buffer.GetEndIter(), true)
	if text == "" {
		return nil
	}
	words, _ := c.Splitter(text, 0)
	return words
}

func (c *Completer) onChange() {
	t, v, blank := State(c.Buffer)
	c.cursor = v

	// If the cursor is on a blank character, then we should not
	// autocomplete anything, so we set the states to nil.
	if blank {
		c.words = nil
		c.index = -1
		c.Popdown()
		return
	}

	c.words, c.index = c.Splitter(t, v)
	c.complete()
}

func (c *Completer) complete() {
	c.Clear()

	var widgets []gtk.IWidget
	if len(c.words) > 0 {
		widgets = c.update()
	}

	if len(widgets) > 0 {
		c.Popover.SetPointingTo(CursorRect(c.Input))
		c.Popup()
	} else {
		c.Popdown()
		return
	}

	for i, widget := range widgets {
		r, _ := gtk.ListBoxRowNew()
		r.Add(widget)
		r.Show()

		c.List.Add(r)

		if i == 0 {
			c.List.SelectRow(r)
		}
	}
}

func (c *Completer) update() []gtk.IWidget {
	// If we don't have a completer, then don't run.
	if c.completer == nil {
		return nil
	}

	c.entries = c.completer.Complete(c.words, c.index)

	var widgets = make([]gtk.IWidget, len(c.entries))

	for i, entry := range c.entries {
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
		if !entry.Secondary.IsEmpty() {
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

			httputil.AsyncImage(context.Background(), img, entry.IconURL, pps...)
		}

		widgets[i] = b
	}

	return widgets
}
