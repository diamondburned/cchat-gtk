package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type PresendMessage struct {
	message.PresendContainer
	Message
}

func NewPresendMessage(msg input.PresendMessage) PresendMessage {
	msgc := message.NewPresendContainer(msg)
	attachCompact(msgc.GenericContainer)

	return PresendMessage{
		PresendContainer: msgc,
		Message:          Message{msgc.GenericContainer},
	}
}

type Message struct {
	*message.GenericContainer
}

var _ container.MessageRow = (*Message)(nil)

func NewMessage(msg cchat.MessageCreate) Message {
	msgc := message.NewContainer(msg)
	attachCompact(msgc)
	message.FillContainer(msgc, msg)

	return Message{msgc}
}

func NewEmptyMessage() Message {
	ct := message.NewEmptyContainer()
	attachCompact(ct)

	return Message{ct}
}

var messageTimeCSS = primitives.PrepareClassCSS("message-time", `
	.message-time {
		margin-left:  1em;
		margin-right: 1em;
	}
`)

var messageAuthorCSS = primitives.PrepareClassCSS("message-author", `
	.message-author {
		margin-right: 0.5em;
	}
`)

func attachCompact(container *message.GenericContainer) {
	container.Timestamp.SetVAlign(gtk.ALIGN_START)
	container.Username.SetMaxWidthChars(25)
	container.Username.SetEllipsize(pango.ELLIPSIZE_NONE)
	container.Username.SetLineWrap(true)
	container.Username.SetLineWrapMode(pango.WRAP_WORD_CHAR)

	messageTimeCSS(container.Timestamp)
	messageAuthorCSS(container.Username)

	container.PackStart(container.Timestamp, false, false, 0)
	container.PackStart(container.Username, false, false, 0)
	container.PackStart(container.Content, true, true, 0)
	container.SetClass("compact")
}
