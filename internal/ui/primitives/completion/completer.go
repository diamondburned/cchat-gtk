package completion

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/diamondburned/cchat/utils/split"
	"github.com/gotk3/gotk3/gtk"
)

type Completeable interface {
	Update([]string, int) []gtk.IWidget
	Word(i int) string
}

type Completer struct {
	ctrl Completeable

	Input   *gtk.TextView
	Buffer  *gtk.TextBuffer
	List    *gtk.ListBox
	Popover *gtk.Popover

	Words  []string
	Index  int
	Cursor int
}

func WrapCompleter(input *gtk.TextView, ctrl Completeable) {
	NewCompleter(input, ctrl)
}

func NewCompleter(input *gtk.TextView, ctrl Completeable) *Completer {
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
		Input:   input,
		Buffer:  ibuf,
		List:    l,
		Popover: p,
		ctrl:    ctrl,
	}

	// This one is for buffer modification.
	ibuf.Connect("end-user-action", c.onChange)
	// This one is for when the cursor moves.
	input.Connect("move-cursor", c.onChange)

	l.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		SwapWord(ibuf, ctrl.Word(r.GetIndex()), c.Cursor)
		c.Clear()
		c.Hide()
		input.GrabFocus()
	})

	return c
}

func (c *Completer) Hide() {
	c.Popover.Popdown()
}

func (c *Completer) Clear() {
	var children = c.List.GetChildren()
	if children.Length() == 0 {
		return
	}

	children.Foreach(func(i interface{}) {
		w := i.(gtk.IWidget).ToWidget()
		c.List.Remove(w)
		w.Destroy()
	})
}

func (c *Completer) onChange() {
	t, v, blank := State(c.Buffer)
	c.Cursor = v

	// If the curssor is on a blank character, then we should not
	// autocomplete anything, so we set the states to nil.
	if blank {
		c.Words = nil
		c.Index = -1
	} else {
		c.Words, c.Index = split.SpaceIndexed(t, v)
	}

	c.complete()
}

func (c *Completer) complete() {
	c.Clear()

	var widgets []gtk.IWidget
	if len(c.Words) > 0 {
		widgets = c.ctrl.Update(c.Words, c.Index)
	}

	if len(widgets) > 0 {
		c.Popover.SetPointingTo(CursorRect(c.Input))
		c.Popover.Popup()
	} else {
		c.Hide()
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
