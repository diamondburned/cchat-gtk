package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
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

	return Message{msgc}
}

func NewEmptyMessage() Message {
	return Message{message.NewEmptyContainer()}
}

func (m Message) Attach() []gtk.IWidget {
	return []gtk.IWidget{m.Timestamp, m.Username, m.Content}
}
