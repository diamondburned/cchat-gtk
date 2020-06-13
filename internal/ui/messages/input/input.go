package input

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/completion"
	"github.com/gotk3/gotk3/gtk"
)

// Controller is an interface to control message containers.
type Controller interface {
	AddPresendMessage(msg PresendMessage) (onErr func(error))
}

type InputView struct {
	*gtk.Box
	*Field
	Completer *completion.View
}

func NewView(ctrl Controller) *InputView {
	text, _ := gtk.TextViewNew()
	text.SetSensitive(false)
	text.SetWrapMode(gtk.WRAP_WORD_CHAR)
	text.SetProperty("top-margin", inputmargin)
	text.SetProperty("left-margin", inputmargin)
	text.SetProperty("right-margin", inputmargin)
	text.SetProperty("bottom-margin", inputmargin)
	text.Show()

	// Bind the text event handler to text first.
	c := completion.New(text)
	c.Show()

	// Bind the input callback later.
	f := NewField(text, ctrl)
	f.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	b.PackStart(c, false, true, 0)
	b.PackStart(f, false, false, 0)
	b.Show()

	// Connect to the field's revealer. On resize, we want the autocompleter to
	// have the right padding too.
	f.username.Connect("size-allocate", func(w gtk.IWidget) {
		// Set the autocompleter's left margin to be the same.
		c.SetMarginStart(w.ToWidget().GetAllocatedWidth())
	})

	return &InputView{b, f, c}
}

func (v *InputView) SetSender(session cchat.Session, sender cchat.ServerMessageSender) {
	v.Field.SetSender(session, sender)

	// Ignore ok; completer can be nil.
	completer, _ := sender.(cchat.ServerMessageSendCompleter)
	v.Completer.SetCompleter(completer)
}

type Field struct {
	*gtk.Box
	username *usernameContainer

	TextScroll *gtk.ScrolledWindow
	text       *gtk.TextView
	buffer     *gtk.TextBuffer

	UserID string
	Sender cchat.ServerMessageSender

	ctrl Controller
}

const inputmargin = 4

func NewField(text *gtk.TextView, ctrl Controller) *Field {
	username := newUsernameContainer()
	username.Show()

	buf, _ := text.GetBuffer()

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Add(text)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetProperty("propagate-natural-height", true)
	sw.SetProperty("max-content-height", 150)
	sw.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(username, false, false, 0)
	box.PackStart(sw, true, true, 0)
	box.Show()

	field := &Field{
		Box:        box,
		username:   username,
		TextScroll: sw,
		text:       text,
		buffer:     buf,
		ctrl:       ctrl,
	}

	text.SetFocusHAdjustment(sw.GetHAdjustment())
	text.SetFocusVAdjustment(sw.GetVAdjustment())
	text.Connect("key-press-event", field.keyDown)

	return field
}

// Reset prepares the field before SetSender() is called.
func (f *Field) Reset() {
	// Paranoia.
	f.text.SetSensitive(false)

	f.UserID = ""
	f.Sender = nil
	f.username.Reset()

	// reset the input
	f.buffer.Delete(f.buffer.GetBounds())
}

// SetSender changes the sender of the input field. If nil, the input will be
// disabled. Reset() should be called first.
func (f *Field) SetSender(session cchat.Session, sender cchat.ServerMessageSender) {
	// Update the left username container in the input.
	f.username.Update(session, sender)

	// Set the sender.
	if sender != nil {
		f.Sender = sender
		f.text.SetSensitive(true)
	}
}

// yankText cuts the text from the input field and returns it.
func (f *Field) yankText() string {
	start, end := f.buffer.GetBounds()

	text, _ := f.buffer.GetText(start, end, false)
	if text != "" {
		f.buffer.Delete(start, end)
	}

	return text
}
