package input

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// Controller is an interface to control message containers.
type Controller interface {
	AddPresendMessage(msg PresendMessage) (onErr func(error))
	LatestMessageFrom(userID string) (messageID string, ok bool)
}

type InputView struct {
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

	// Bind the input callback later.
	f := NewField(text, ctrl)
	f.Show()

	// // Connect to the field's revealer. On resize, we want the autocompleter to
	// // have the right padding too.
	// f.username.Connect("size-allocate", func(w gtk.IWidget) {
	// 	// Set the autocompleter's left margin to be the same.
	// 	c.SetMarginStart(w.ToWidget().GetAllocatedWidth())
	// })

	return &InputView{f, c}
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
	editor cchat.ServerMessageEditor

	ctrl Controller

	// editing state
	editingID string // never empty
}

const inputmargin = 4

func NewField(text *gtk.TextView, ctrl Controller) *Field {
	username := newUsernameContainer()
	username.Show()

	buf, _ := text.GetBuffer()

	sw := scrollinput.NewV(text, 150)
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
	f.editor = nil
	f.username.Reset()

	// reset the input
	f.buffer.Delete(f.buffer.GetBounds())
}

// SetSender changes the sender of the input field. If nil, the input will be
// disabled. Reset() should be called first.
func (f *Field) SetSender(session cchat.Session, sender cchat.ServerMessageSender) {
	// Update the left username container in the input.
	f.username.Update(session, sender)
	f.UserID = session.ID()

	// Set the sender.
	if sender != nil {
		f.Sender = sender
		f.text.SetSensitive(true)

		// Allow editor to be nil.
		ed, ok := sender.(cchat.ServerMessageEditor)
		if !ok {
			log.Printlnf("Editor is not implemented for %T", sender)
		}
		f.editor = ed
	}
}

// Editable returns whether or not the input field can be edited.
func (f *Field) Editable(msgID string) bool {
	return f.editor != nil && f.editor.MessageEditable(msgID)
}

func (f *Field) StartEditing(msgID string) bool {
	// Do we support message editing? If not, exit.
	if !f.Editable(msgID) {
		return false
	}

	// Try and request the old message content for editing.
	content, err := f.editor.RawMessageContent(msgID)
	if err != nil {
		// TODO: show error
		log.Error(errors.Wrap(err, "Failed to get message content"))
		return false
	}

	// Set the current editing state and set the input after requesting the
	// content.
	f.editingID = msgID
	f.buffer.SetText(content)

	return true
}

// StopEditing cancels the current editing message. It returns a false and does
// nothing if the editor is not editing anything.
func (f *Field) StopEditing() bool {
	if f.editingID == "" {
		return false
	}

	f.editingID = ""
	f.clearText()

	return true
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

// clearText wipes the input field
func (f *Field) clearText() {
	f.buffer.Delete(f.buffer.GetBounds())
}

// getText returns the text from the input, but it doesn't cut it.
func (f *Field) getText() string {
	start, end := f.buffer.GetBounds()
	text, _ := f.buffer.GetText(start, end, false)
	return text
}

func (f *Field) textLen() int {
	return f.buffer.GetCharCount()
}
