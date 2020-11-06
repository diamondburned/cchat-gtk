package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
)

type Container struct {
	*container.GridContainer
}

func NewContainer(ctrl container.Controller) *Container {
	c := container.NewGridContainer(constructor{}, ctrl)
	primitives.AddClass(c, "compact-conatainer")
	return &Container{c}
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		c.GridContainer.CreateMessageUnsafe(msg)
		c.GridContainer.CleanMessages()
	})
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() { c.GridContainer.UpdateMessageUnsafe(msg) })
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() { c.GridContainer.DeleteMessageUnsafe(msg) })
}

type constructor struct{}

func (constructor) NewMessage(msg cchat.MessageCreate) container.GridMessage {
	return NewMessage(msg)
}

func (constructor) NewPresendMessage(msg input.PresendMessage) container.PresendGridMessage {
	return NewPresendMessage(msg)
}
