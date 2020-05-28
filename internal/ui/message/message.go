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
	index int
	ID    string
	Nonce string

	Timestamp *gtk.Label
	Username  *gtk.Label
	Content   *gtk.Label
}

func NewMessage(msg cchat.MessageCreate) Message {
	ts, _ := gtk.LabelNew("")
	ts.SetLineWrap(true)
	ts.SetLineWrapMode(pango.WRAP_WORD)
	ts.SetHAlign(gtk.ALIGN_END)
	ts.SetVAlign(gtk.ALIGN_START)
	ts.Show()

	user, _ := gtk.LabelNew("")
	user.SetLineWrap(true)
	user.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	user.SetHAlign(gtk.ALIGN_END)
	user.SetVAlign(gtk.ALIGN_START)
	user.Show()

	content, _ := gtk.LabelNew("")
	content.SetHExpand(true)
	content.SetXAlign(0) // left-align with size filled
	content.SetVAlign(gtk.ALIGN_START)
	content.SetLineWrap(true)
	content.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	content.Show()

	m := Message{
		ID:        msg.ID(),
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

func (m *Message) Attach(grid *gtk.Grid, row int) {
	grid.Attach(m.Timestamp, 0, row, 1, 1)
	grid.Attach(m.Username, 1, row, 1, 1)
	grid.Attach(m.Content, 2, row, 1, 1)
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
