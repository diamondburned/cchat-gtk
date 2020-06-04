package compact

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Message struct {
	index    int
	ID       string
	AuthorID string
	Nonce    string

	Timestamp *gtk.Label
	Username  *gtk.Label
	Content   *gtk.Label
}

func NewMessage(msg cchat.MessageCreate) Message {
	m := NewEmptyMessage()
	m.ID = msg.ID()
	m.UpdateTimestamp(msg.Time())
	m.UpdateAuthor(msg.Author())
	m.UpdateContent(msg.Content())

	if noncer, ok := msg.(cchat.MessageNonce); ok {
		m.Nonce = noncer.Nonce()
	}

	return m
}

func NewPresendMessage(content string, author text.Rich, authorID, nonce string) Message {
	msgc := NewEmptyMessage()
	msgc.Nonce = nonce
	msgc.AuthorID = authorID
	msgc.SetSensitive(false)
	msgc.UpdateContent(text.Rich{Content: content})
	msgc.UpdateTimestamp(time.Now())
	msgc.updateAuthorName(author)

	return msgc
}

func NewEmptyMessage() Message {
	ts, _ := gtk.LabelNew("")
	ts.SetLineWrap(true)
	ts.SetLineWrapMode(pango.WRAP_WORD)
	ts.SetHAlign(gtk.ALIGN_END)
	ts.SetVAlign(gtk.ALIGN_START)
	ts.SetSelectable(true)
	ts.Show()

	user, _ := gtk.LabelNew("")
	user.SetMaxWidthChars(35)
	user.SetLineWrap(true)
	user.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	user.SetHAlign(gtk.ALIGN_END)
	user.SetVAlign(gtk.ALIGN_START)
	user.SetSelectable(true)
	user.Show()

	content, _ := gtk.LabelNew("")
	content.SetHExpand(true)
	content.SetXAlign(0) // left-align with size filled
	content.SetVAlign(gtk.ALIGN_START)
	content.SetLineWrap(true)
	content.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	content.SetSelectable(true)
	content.Show()

	return Message{
		Timestamp: ts,
		Username:  user,
		Content:   content,
	}
}

func (m *Message) SetSensitive(sensitive bool) {
	m.Timestamp.SetSensitive(sensitive)
	m.Username.SetSensitive(sensitive)
	m.Content.SetSensitive(sensitive)
}

func (m *Message) Attach(grid *gtk.Grid, row int) {
	grid.Attach(m.Timestamp, 0, row, 1, 1)
	grid.Attach(m.Username, 1, row, 1, 1)
	grid.Attach(m.Content, 2, row, 1, 1)
}

func (m *Message) UpdateTimestamp(t time.Time) {
	m.Timestamp.SetLabel(humanize.TimeAgo(t))
	m.Timestamp.SetTooltipText(t.Format(time.Stamp))
}

func (m *Message) UpdateAuthor(author cchat.MessageAuthor) {
	m.AuthorID = author.ID()
	m.updateAuthorName(author.Name())
}

func (m *Message) updateAuthorName(name text.Rich) {
	m.Username.SetLabel(name.Content)
	m.Username.SetTooltipText(name.Content)
}

func (m *Message) UpdateContent(content text.Rich) {
	m.Content.SetLabel(content.Content)
}
