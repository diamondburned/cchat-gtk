package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/gotk3/gotk3/gtk"
)

type PresendMessage struct {
	*message.GenericPresendContainer
}

func NewPresendMessage(msg input.PresendMessage) PresendMessage {
	return PresendMessage{
		GenericPresendContainer: message.NewPresendContainer(msg),
	}
}

func (p PresendMessage) Attach(grid *gtk.Grid, row int) {
	attachGenericContainer(p.GenericContainer, grid, row)
}

type Message struct {
	*message.GenericContainer
}

var _ container.GridMessage = (*Message)(nil)

func NewMessage(msg cchat.MessageCreate) Message {
	return Message{
		GenericContainer: message.NewContainer(msg),
	}
}

func NewEmptyMessage() Message {
	return Message{
		GenericContainer: message.NewEmptyContainer(),
	}
}

// TODO: fix a bug here related to new messages overlapping
func (m Message) Attach(grid *gtk.Grid, row int) {
	attachGenericContainer(m.GenericContainer, grid, row)
}

func attachGenericContainer(m *message.GenericContainer, grid *gtk.Grid, row int) {
	grid.Attach(m.Timestamp, 0, row, 1, 1)
	grid.Attach(m.Username, 1, row, 1, 1)
	grid.Attach(m.Content, 2, row, 1, 1)
}
