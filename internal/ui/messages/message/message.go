package message

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Container interface {
	ID() string
	AuthorID() string
	Nonce() string

	UpdateAuthor(cchat.MessageAuthor)
	UpdateAuthorName(text.Rich)
	UpdateContent(text.Rich)
	UpdateTimestamp(time.Time)
}

func FillContainer(c Container, msg cchat.MessageCreate) {
	c.UpdateAuthor(msg.Author())
	c.UpdateContent(msg.Content())
	c.UpdateTimestamp(msg.Time())
}

// GenericContainer provides a single generic message container for subpackages
// to use.
type GenericContainer struct {
	id       string
	authorID string
	nonce    string

	Timestamp *gtk.Label
	Username  *gtk.Label
	Content   *gtk.Label
}

var _ Container = (*GenericContainer)(nil)

// NewContainer creates a new message container with the given ID and nonce. It
// does not update the widgets, so FillContainer should be called afterwards.
func NewContainer(msg cchat.MessageCreate) *GenericContainer {
	c := NewEmptyContainer()
	c.id = msg.ID()
	c.authorID = msg.Author().ID()

	if noncer, ok := msg.(cchat.MessageNonce); ok {
		c.nonce = noncer.Nonce()
	}

	return c
}

func NewEmptyContainer() *GenericContainer {
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

	return &GenericContainer{
		Timestamp: ts,
		Username:  user,
		Content:   content,
	}
}

func (m *GenericContainer) ID() string {
	return m.id
}

func (m *GenericContainer) AuthorID() string {
	return m.authorID
}

func (m *GenericContainer) Nonce() string {
	return m.nonce
}

func (m *GenericContainer) UpdateTimestamp(t time.Time) {
	m.Timestamp.SetLabel(humanize.TimeAgo(t))
	m.Timestamp.SetTooltipText(t.Format(time.Stamp))
}

func (m *GenericContainer) UpdateAuthor(author cchat.MessageAuthor) {
	m.UpdateAuthorName(author.Name())
}

func (m *GenericContainer) UpdateAuthorName(name text.Rich) {
	m.Username.SetMarkup(parser.RenderMarkup(name))
}

func (m *GenericContainer) UpdateContent(content text.Rich) {
	m.Content.SetMarkup(parser.RenderMarkup(content))
}
