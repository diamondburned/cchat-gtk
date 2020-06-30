package commander

import (
	"fmt"
	"io"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/autoscroll"
	"github.com/google/shlex"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type SessionCommander interface {
	cchat.Session
	cchat.Commander
}

type Buffer struct {
	*gtk.TextBuffer
	svcname string
	cmder   SessionCommander
}

// NewBuffer creates a new buffer with the given SessionCommander, or returns
// nil if cmder is nil.
func NewBuffer(svc cchat.Service, cmder SessionCommander) *Buffer {
	if cmder == nil {
		return nil
	}

	b, _ := gtk.TextBufferNew(nil)
	b.CreateTag("error", map[string]interface{}{
		"foreground": "#FF0000",
	})
	return &Buffer{b, svc.Name().Content, cmder}
}

// WriteError is not thread-safe.
func (b *Buffer) WriteError(err error) {
	b.InsertWithTagByName(b.GetEndIter(), err.Error()+"\n", "error")
}

// WriteUnsafe is not thread-safe.
func (b *Buffer) WriteUnsafe(bytes []byte) {
	b.Insert(b.GetEndIter(), string(bytes))
}

// Printlnf is not thread-safe.
func (b *Buffer) Printlnf(f string, v ...interface{}) {
	b.WriteUnsafe([]byte(fmt.Sprintf(f+"\n", v...)))
}

// Write is thread-safe.
func (b *Buffer) Write(bytes []byte) (int, error) {
	gts.ExecAsync(func() { b.WriteUnsafe(bytes) })
	return len(bytes), nil
}

func (b *Buffer) ShowDialog() {
	SpawnDialog(b)
}

var entryCSS = primitives.PrepareCSS(`
	* {
		font-family: monospace;
		border-radius: 0;
	}
`)

type Session struct {
	*gtk.Box
	words  []string
	cmder  cchat.Commander
	buffer *Buffer
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
	v, _ := gtk.TextViewNewWithBuffer(buf.TextBuffer)
	v.SetEditable(false)
	v.SetProperty("monospace", true)
	v.SetBorderWidth(8)
	v.SetPixelsAboveLines(1)
	v.SetWrapMode(gtk.WRAP_WORD_CHAR)
	v.Show()

	s := autoscroll.NewScrolledWindow()
	s.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	s.Add(v)
	s.Show()

	i, _ := gtk.EntryNew()
	primitives.AttachCSS(i, entryCSS)
	i.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.PackStart(s, true, true, 0)
	b.PackStart(i, false, false, 0)

	session := &Session{
		Box:    b,
		cmder:  cmder,
		buffer: buf,
	}

	i.Connect("activate", session.inputActivate)
	// Split words on typing to provide live errors.
	i.Connect("changed", func(i *gtk.Entry) {
		t, _ := i.GetText()

		w, err := shlex.Split(t)
		if err != nil {
			i.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "dialog-error")
			i.SetIconTooltipText(gtk.ENTRY_ICON_SECONDARY, err.Error())
			session.words = nil
		} else {
			i.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "")
			session.words = w
		}
	})

	// Focus on the input by default.
	i.GrabFocus()

	return session
}

func (s *Session) inputActivate(e *gtk.Entry) {
	// If the input is empty, then ignore.
	if len(s.words) == 0 {
		return
	}

	r, err := s.cmder.RunCommand(s.words)
	if err != nil {
		s.buffer.WriteError(err)
		return
	}

	// Clear the entry.
	e.SetText("")

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
}
