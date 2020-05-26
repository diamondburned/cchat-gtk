package message

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Message struct {
	ID    string
	Nonce string

	*gtk.Box
	Timestamp *gtk.Label
	Username  *gtk.Label
	Content   *gtk.Label
}

func NewMessage(msg cchat.MessageCreate) Message {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 3)
	box.Show()

	ts, _ := gtk.LabelNew("")
	ts.Show()
	ts.SetWidthChars(12)
	ts.SetLineWrapMode(pango.WRAP_WORD)

	user, _ := gtk.LabelNew("")
	user.Show()
	user.SetLineWrapMode(pango.WRAP_WORD_CHAR)

	content, _ := gtk.LabelNew("")
	content.Show()

	box.PackStart(ts, false, false, 0)
	box.PackStart(user, false, false, 0)
	box.PackStart(content, true, true, 0)

	m := Message{
		ID:        msg.ID(),
		Box:       box,
		Timestamp: ts,
		Username:  user,
		Content:   content,
	}
	m.UpdateTimestamp(msg.Time())
	m.UpdateAuthor(msg.Author())
	m.UpdateContent(msg.Content())

	if noncer, ok := msg.(cchat.MessageNonce); ok {
		m.Nonce = noncer.Nonce()
	}

	return m
}

func (m *Message) UpdateTimestamp(t time.Time) {
	m.Timestamp.SetLabel(humanize.TimeAgo(t))
}

func (m *Message) UpdateAuthor(author text.Rich) {
	m.Username.SetLabel(author.Content)
}

func (m *Message) UpdateContent(content text.Rich) {
	m.Content.SetLabel(content.Content)
}
