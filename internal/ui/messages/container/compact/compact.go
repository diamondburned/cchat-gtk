package compact

import (
	"github.com/diamondburned/cchat"
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

type constructor struct{}

func (constructor) NewMessage(msg cchat.MessageCreate) container.GridMessage {
	return NewMessage(msg)
}

func (constructor) NewPresendMessage(msg input.PresendMessage) container.PresendGridMessage {
	return NewPresendMessage(msg)
}
