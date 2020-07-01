package commander

import (
	"fmt"
	"io"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

var monospace = primitives.PrepareCSS(`
	* {
		font-family: monospace;
		border-radius: 0;
	}
`)

type Session struct {
	*gtk.Box

	cmder  cchat.Commander
	buffer *Buffer
	cmplt  *completer

	inputbuf *gtk.TextBuffer

	// words []string
	// index int
}

func SpawnDialog(buf *Buffer) {
	s := NewSession(buf.cmder, buf)
	s.Show()

	h, _ := gtk.HeaderBarNew()
	h.SetTitle(fmt.Sprintf(
		"Commander: %s on %s",
		buf.cmder.Name().Content, buf.svcname,
	))
	h.SetShowCloseButton(true)
	h.Show()

	d, _ := gts.NewEmptyModalDialog()
	d.SetDefaultSize(450, 250)
	d.SetTitlebar(h)
	d.Add(s)
	d.Show()
}

func NewSession(cmder cchat.Commander, buf *Buffer) *Session {
	view, _ := gtk.TextViewNewWithBuffer(buf.TextBuffer)
	view.SetEditable(false)
	view.SetProperty("monospace", true)
	view.SetPixelsAboveLines(1)
	view.SetWrapMode(gtk.WRAP_WORD_CHAR)
	view.Show()

	scroll := autoscroll.NewScrolledWindow()
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.Add(view)
	scroll.Show()

	input, _ := gtk.TextViewNew()
	input.SetSizeRequest(-1, 35) // magic height 35px
	primitives.AttachCSS(input, monospace)
	input.Show()

	inputbuf, _ := input.GetBuffer()

	inputscroll := scrollinput.NewH(input)
	inputscroll.Show()

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	sep.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.PackStart(scroll, true, true, 0)
	b.PackStart(sep, false, false, 0)
	b.PackStart(inputscroll, false, false, 0)

	session := &Session{
		Box:      b,
		cmder:    cmder,
		buffer:   buf,
		cmplt:    newCompleter(input, cmder),
		inputbuf: inputbuf,
	}

	input.Connect("key-press-event", session.inputActivate)
	input.GrabFocus()

	primitives.AddClass(b, "commander")
	primitives.AddClass(view, "command-buffer")
	primitives.AddClass(input, "command-input")

	return session
}

func (s *Session) inputActivate(v *gtk.TextView, ev *gdk.Event) bool {
	// If the keypress is not enter, then ignore.
	if kev := gdk.EventKeyNewFromEvent(ev); kev.KeyVal() != gdk.KEY_Return {
		return false
	}

	// If the input is empty, then ignore.
	if len(s.cmplt.Words) == 0 {
		return true
	}

	r, err := s.cmder.RunCommand(s.cmplt.Words)
	if err != nil {
		s.buffer.WriteError(err)
		return true
	}

	// Clear the entry.
	s.inputbuf.Delete(s.inputbuf.GetBounds())

	var then = time.Now()
	s.buffer.Printlnf("%s: Running command...", then.Format(time.Kitchen))

	go func() {
		_, err := io.Copy(s.buffer, r)
		r.Close()

		gts.ExecAsync(func() {
			if err != nil {
				s.buffer.WriteError(errors.Wrap(err, "Internal error"))
			}

			var now = time.Now()
			s.buffer.Printlnf(
				"%s: Finished running command, took %s.",
				now.Format(time.Kitchen),
				now.Sub(then).String(),
			)
		})
	}()

	return true
}

type completer struct {
	*completion.Completer

	completer cchat.CommandCompleter
	choices   []string
}

func newCompleter(input *gtk.TextView, v cchat.Commander) *completer {
	completer := &completer{}
	completer.Completer = completion.NewCompleter(input, completer)

	c, ok := v.(cchat.CommandCompleter)
	if ok {
		completer.completer = c
	}

	return completer
}

func (c *completer) Update(words []string, offset int) []gtk.IWidget {
	if c.completer == nil {
		return nil
	}

	c.choices = c.completer.CompleteCommand(words, offset)
	var widgets = make([]gtk.IWidget, 0, len(c.choices))

	for _, choice := range c.choices {
		l, _ := gtk.LabelNew(choice)
		l.SetXAlign(0)
		l.SetEllipsize(pango.ELLIPSIZE_END)
		primitives.AttachCSS(l, monospace)
		l.Show()

		widgets = append(widgets, l)
	}

	return widgets
}

func (c *completer) Word(i int) string {
	return c.choices[i]
}
