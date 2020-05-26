package input

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Field struct {
	*gtk.ScrolledWindow
	text   *gtk.TextView
	buffer *gtk.TextBuffer

	sender cchat.ServerMessageSender
}

func NewField() *Field {
	text, _ := gtk.TextViewNew()
	text.Show()

	buf, _ := text.GetBuffer()

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Show()
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetProperty("max-content-height", 150)
	sw.Add(text)

	return &Field{
		sw,
		text,
		buf,
		nil,
	}
}

// SetSender changes the sender of the input field. If nil, the input will be
// disabled.
func (f *Field) SetSender(sender cchat.ServerMessageSender) {
	f.sender = sender
	f.text.SetSensitive(sender != nil) // grey if sender is nil

	// reset the input
	f.buffer.Delete(f.buffer.GetBounds())
}

// SendMessage yanks the text from the input field and sends it to the backend.
// This function is not thread-safe.
func (f *Field) SendMessage() {
	if f.sender == nil {
		return
	}

	var text = f.yankText()
	if text == "" {
		return
	}

	var sender = f.sender

	go func() {
		if err := sender.SendMessage(SendMessageData(text)); err != nil {
			log.Error(errors.Wrap(err, "Failed to send message"))
		}
	}()
}

type SendMessageData string

func (s SendMessageData) Content() string { return string(s) }

// yankText cuts the text from the input field and returns it.
func (f *Field) yankText() string {
	start, end := f.buffer.GetBounds()

	text, _ := f.buffer.GetText(start, end, false)
	if text != "" {
		f.buffer.Delete(start, end)
	}

	return text
}
