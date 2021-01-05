package compact

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var messageTimeCSS = primitives.PrepareClassCSS("", `
	.message-time {
		margin-left:  1em;
		margin-right: 1em;
	}
`)

var messageAuthorCSS = primitives.PrepareClassCSS("", `
	.message-author {
		margin-right: 0.5em;
	}
`)

type PresendMessage struct {
	message.PresendContainer
	Message
}

func NewPresendMessage(msg input.PresendMessage) PresendMessage {
	msgc := message.NewPresendContainer(msg)

	return PresendMessage{
		PresendContainer: msgc,
		Message:          wrapMessage(msgc.GenericContainer),
	}
}

type Message struct {
	*message.GenericContainer
	Timestamp *gtk.Label
	Username  *labeluri.Label
}

var _ container.MessageRow = (*Message)(nil)

func NewMessage(msg cchat.MessageCreate) Message {
	msgc := wrapMessage(message.NewContainer(msg))
	message.FillContainer(msgc, msg)
	return msgc
}

func NewEmptyMessage() Message {
	ct := message.NewEmptyContainer()
	return wrapMessage(ct)
}

func wrapMessage(ct *message.GenericContainer) Message {
	ts := message.NewTimestamp()
	ts.SetVAlign(gtk.ALIGN_START)
	ts.Show()
	messageTimeCSS(ts)

	user := message.NewUsername()
	user.SetMaxWidthChars(25)
	user.SetEllipsize(pango.ELLIPSIZE_NONE)
	user.SetLineWrap(true)
	user.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	user.Show()
	messageAuthorCSS(user)

	ct.PackStart(ts, false, false, 0)
	ct.PackStart(user, false, false, 0)
	ct.PackStart(ct.Content, true, true, 0)
	ct.SetClass("compact")

	return Message{
		GenericContainer: ct,
		Timestamp:        ts,
		Username:         user,
	}
}

// SetReferenceHighlighter sets the reference highlighter into the message.
func (m Message) SetReferenceHighlighter(r labeluri.ReferenceHighlighter) {
	m.GenericContainer.SetReferenceHighlighter(r)
	m.Username.SetReferenceHighlighter(r)
}

func (m Message) UpdateTimestamp(t time.Time) {
	m.GenericContainer.UpdateTimestamp(t)
	m.Timestamp.SetText(humanize.TimeAgo(t))
	m.Timestamp.SetTooltipText(t.Format(time.Stamp))
}

func (m Message) UpdateAuthor(author cchat.Author) {
	m.GenericContainer.UpdateAuthor(author)

	cfg := markup.RenderConfig{}
	cfg.NoReferencing = true
	cfg.SetForegroundAnchor(m.ContentBodyStyle)

	m.Username.SetOutput(markup.RenderCmplxWithConfig(author.Name(), cfg))
}
