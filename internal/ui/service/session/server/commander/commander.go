package commander

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/diamondburned/cchat/utils/split"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
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
	cmplt  *completion.Completer
	buffer *Buffer

	inputbuf *gtk.TextBuffer

	// words []string
	// index int
}

func SpawnDialog(buf *Buffer) {
	s := NewSession(buf)
	s.Show()

	h, _ := gtk.HeaderBarNew()
	h.SetShowCloseButton(true)
	h.Show()

	rm := buf.name.OnUpdate(func() {
		h.SetTitle("Commander: " + buf.name.Label().Content)
	})
	h.Connect("destroy", rm)

	d, _ := gts.NewEmptyModalDialog()
	d.SetDefaultSize(450, 250)
	d.SetTitlebar(h)
	d.Add(s)
	d.Show()
}

func NewSession(buf *Buffer) *Session {
	view, _ := gtk.TextViewNewWithBuffer(buf.TextBuffer)
	view.SetEditable(false)
	view.SetProperty("monospace", true)
	view.SetPixelsAboveLines(1)
	view.SetWrapMode(gtk.WRAP_WORD_CHAR)
	view.SetBorderWidth(8)
	view.Show()

	scroll := autoscroll.NewScrolledWindow()
	scroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	scroll.Add(view)
	scroll.Show()

	input, _ := gtk.TextViewNew()
	input.SetSizeRequest(-1, 35) // magic height 35px
	input.SetBorderWidth(8)
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

	completer := completion.NewCompleter(input)
	completer.Splitter = split.ArgsIndexed
	completer.SetCompleter(buf.cmder.AsCompleter())

	session := &Session{
		Box:      b,
		cmder:    buf.cmder,
		cmplt:    completer,
		buffer:   buf,
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

	// Get the slice of words.
	var words = s.cmplt.Content()
	// If the input is empty, then ignore.
	if len(words) == 0 {
		return true
	}

	// Clear the entry.
	s.inputbuf.Delete(s.inputbuf.GetBounds())

	var then = time.Now()
	s.buffer.Systemlnf("%s > %q", then.Format(time.Kitchen), words)

	go func() {
		out, err := s.cmder.Run(words)

		gts.ExecAsync(func() {
			if out != nil {
				s.buffer.WriteOutput(out)
			}

			if err != nil {
				s.buffer.WriteError(err)
			}

			var now = time.Now()
			s.buffer.Systemlnf(
				"%s: Finished running command, took %s.",
				now.Format(time.Kitchen),
				now.Sub(then).String(),
			)
		})
	}()

	return true
}
