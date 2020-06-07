package messages

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container/cozy"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Container interface {
	gtk.IWidget
	cchat.MessagesContainer

	Reset()
	ScrollToBottom()

	// PresendMessage is for unsent messages.
	PresendMessage(input.PresendMessage) (done func(sendError error))
}

type View struct {
	*gtk.Box
	Container container.Container
	SendInput *input.Field

	current cchat.ServerMessage
	author  string
}

func NewView() *View {
	view := &View{}

	// TODO: change
	// view.Container = compact.NewContainer()
	view.Container = cozy.NewContainer()
	view.SendInput = input.NewField(view)

	view.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	view.Box.PackStart(view.Container, true, true, 0)
	view.Box.PackStart(view.SendInput, false, false, 0)
	view.Box.Show()

	return view
}

// JoinServer is not thread-safe, but it calls backend functions asynchronously.
func (v *View) JoinServer(session cchat.Session, server cchat.ServerMessage) {
	if v.current != nil {
		// Backend should handle synchronizing joins and leaves if it needs to.
		go func() {
			if err := v.current.LeaveServer(); err != nil {
				log.Error(errors.Wrap(err, "Error leaving server"))
			}
		}()

		// Clean all messages.
		v.Container.Reset()
	}

	v.current = server

	// Skipping ok check because sender can be nil. Without the empty check, Go
	// will panic.
	sender, _ := server.(cchat.ServerMessageSender)
	v.SendInput.SetSender(session, sender)

	go func() {
		if err := v.current.JoinServer(v.Container); err != nil {
			log.Error(errors.Wrap(err, "Failed to join server"))
		}
	}()
}

func (v *View) PresendMessage(msg input.PresendMessage) func(error) {
	return v.Container.PresendMessage(msg)
}
