package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
)

type Container struct {
	*container.GridContainer
}

func NewContainer() *Container {
	return &Container{
		GridContainer: container.NewGridContainer(constructor{}),
	}
}

type constructor struct{}

func (constructor) NewMessage(msg cchat.MessageCreate) container.GridMessage {
	return NewMessage(msg)
}

func (constructor) NewPresendMessage(msg input.PresendMessage) container.PresendGridMessage {
	return NewPresendMessage(msg)
}
