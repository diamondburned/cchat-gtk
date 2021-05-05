package compact

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*container.ListContainer
	sg SizeGroups
}

type SizeGroups struct {
	Timestamp *gtk.SizeGroup
	Username  *gtk.SizeGroup
}

func NewSizeGroups() SizeGroups {
	sg1, _ := gtk.SizeGroupNew(gtk.SIZE_GROUP_HORIZONTAL)
	sg2, _ := gtk.SizeGroupNew(gtk.SIZE_GROUP_HORIZONTAL)

	return SizeGroups{sg1, sg2}
}

func (sgs *SizeGroups) Add(msg Message) {
	sgs.Timestamp.AddWidget(msg.Timestamp)
	sgs.Username.AddWidget(msg.Username)
}

var _ container.Container = (*Container)(nil)

func NewContainer(ctrl container.Controller) *Container {
	c := container.NewListContainer(ctrl)
	primitives.AddClass(c, "compact-container")
	return &Container{c, NewSizeGroups()}
}

func (c *Container) NewPresendMessage(state *message.PresendState) container.PresendMessageRow {
	msg := WrapPresendMessage(state)
	c.sg.Add(msg.Message)
	c.addMessage(msg)
	return msg
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		msg := WrapMessage(message.NewState(msg))
		c.sg.Add(msg)
		c.addMessage(msg)
		c.CleanMessages()
	})
}

func (c *Container) addMessage(msg container.MessageRow) {
	_, at := container.InsertPosition(c, msg.Unwrap().Time)
	c.AddMessageAt(msg, at)
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() { container.UpdateMessage(c, msg) })
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() { c.PopMessage(msg.ID()) })
}
