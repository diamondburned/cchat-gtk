package cozy

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/gotk3/gotk3/gtk"
)

type Message interface {
	gtk.IWidget
	message.Container
}

type FullMessage struct {
	*gtk.Box

	Avatar *gtk.Image
	*message.GenericContainer
}

func NewFullMessage()
