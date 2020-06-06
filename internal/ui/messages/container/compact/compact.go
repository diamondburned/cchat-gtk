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
	c := &Container{}
	c.GridContainer = container.NewGridContainer(c)
	return c
}

func (c *Container) NewMessage(msg cchat.MessageCreate) container.GridMessage {
	return NewMessage(msg)
}

func (c *Container) NewPresendMessage(msg input.PresendMessage) container.PresendGridMessage {
	return NewPresendMessage(msg)
}
