package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
)

type Container struct {
	*container.ListContainer
}

func NewContainer(ctrl container.Controller) *Container {
	c := container.NewListContainer(constructor{}, ctrl)
	primitives.AddClass(c, "compact-conatainer")
	return &Container{c}
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		c.ListContainer.CreateMessageUnsafe(msg)
		c.ListContainer.CleanMessages()
	})
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() { c.ListContainer.UpdateMessageUnsafe(msg) })
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() { c.ListContainer.DeleteMessageUnsafe(msg) })
}

type constructor struct{}

func (constructor) NewMessage(msg cchat.MessageCreate) container.MessageRow {
	return NewMessage(msg)
}

func (constructor) NewPresendMessage(msg input.PresendMessage) container.PresendMessageRow {
	return NewPresendMessage(msg)
}
