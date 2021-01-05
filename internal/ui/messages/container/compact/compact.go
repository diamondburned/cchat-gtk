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
	c := container.NewListContainer(ctrl, constructors)
	primitives.AddClass(c, "compact-container")
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

var constructors = container.Constructor{
	NewMessage:        newMessage,
	NewPresendMessage: newPresendMessage,
}

func newMessage(
	msg cchat.MessageCreate, _ container.MessageRow) container.MessageRow {

	return NewMessage(msg)
}

func newPresendMessage(
	msg input.PresendMessage, _ container.MessageRow) container.PresendMessageRow {

	return NewPresendMessage(msg)
}
