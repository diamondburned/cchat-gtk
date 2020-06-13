package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

type PresendMessage struct {
	message.PresendContainer
	Message
}

func NewPresendMessage(msg input.PresendMessage) PresendMessage {
	var msgc = message.NewPresendContainer(msg)

	return PresendMessage{
		PresendContainer: msgc,
		Message:          Message{msgc.GenericContainer},
	}
}

type Message struct {
	*message.GenericContainer
}

var _ container.GridMessage = (*Message)(nil)

func NewMessage(msg cchat.MessageCreate) Message {
	msgc := message.NewContainer(msg)
	message.FillContainer(msgc, msg)

	primitives.AddClass(msgc.Timestamp, "compact-timestamp")
	primitives.AddClass(msgc.Username, "compact-username")
	primitives.AddClass(msgc.Content, "compact-content")

	return Message{msgc}
}

func NewEmptyMessage() Message {
	return Message{message.NewEmptyContainer()}
}

func (m Message) Attach(grid *gtk.Grid, row int) {
	container.AttachRow(grid, row, m.Timestamp, m.Username, m.Content)
}
