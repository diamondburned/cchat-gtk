package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
)

type Container struct {
	*container.ListContainer
}

var _ container.Container = (*Container)(nil)

func NewContainer(ctrl container.Controller) *Container {
	c := container.NewListContainer(ctrl)
	primitives.AddClass(c, "compact-container")
	return &Container{c}
}

func (c *Container) NewPresendMessage(state *message.PresendState) container.PresendMessageRow {
	msg := WrapPresendMessage(state)
	c.AddMessage(msg)
	return msg
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		msg := WrapMessage(message.NewState(msg))
		c.ListContainer.AddMessage(msg)
	})
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() { container.UpdateMessage(c, msg) })
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() { c.PopMessage(msg.ID()) })
}
