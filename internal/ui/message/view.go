package message

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/message/input"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type View struct {
	*gtk.Box
	Container *Container
	SendInput *input.Field

	current cchat.ServerMessage
}

func NewView() *View {
	container := NewContainer()
	sendinput := input.NewField()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(container, true, true, 0)
	box.PackStart(sendinput, false, false, 0)
	box.Show()

	return &View{
		Box:       box,
		Container: container,
		SendInput: sendinput,
	}
}

func (v *View) JoinServer(server cchat.ServerMessage) {
	if v.current != nil {
		if err := v.current.LeaveServer(); err != nil {
			log.Error(errors.Wrap(err, "Error leaving server"))
		}

		// Clean all messages.
		v.Container.Reset()
	}

	v.current = server

	// Skipping ok check because sender can be nil. Without the empty check, Go
	// will panic.
	sender, _ := server.(cchat.ServerMessageSender)
	v.SendInput.SetSender(sender)

	if err := v.current.JoinServer(v.Container); err != nil {
		log.Error(errors.Wrap(err, "Failed to join server"))
	}
}
