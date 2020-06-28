package messages

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container/compact"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container/cozy"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/sadface"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	cozyMessage int = iota
	compactMessage
)

var msgIndex = cozyMessage

func init() {
	config.AppearanceAdd("Message Display", config.Combo(
		&msgIndex, // 0 or 1
		[]string{"Cozy", "Compact"},
		nil,
	))
}

type View struct {
	*sadface.FaceView
	Box *gtk.Box

	InputView *input.InputView
	Container container.Container
	contType  int // msgIndex

	// Inherit some useful methods.
	state
}

func NewView() *View {
	view := &View{}
	view.InputView = input.NewView(view)

	view.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	view.Box.PackEnd(view.InputView, false, false, 0)
	view.Box.Show()

	// Create the message container, which will use PackEnd to add the widget on
	// TOP of the input view.
	view.createMessageContainer()

	// placeholder logo
	logo, _ := gtk.ImageNewFromPixbuf(icons.Logo256())
	logo.Show()

	view.FaceView = sadface.New(view.Box, logo)
	return view
}

func (v *View) createMessageContainer() {
	// Remove the old message container.
	if v.Container != nil {
		v.Box.Remove(v.Container)
	}

	// Update the container type.
	switch v.contType = msgIndex; msgIndex {
	case cozyMessage:
		v.Container = cozy.NewContainer(v)
	case compactMessage:
		v.Container = compact.NewContainer(v)
	}

	// Add the new message container.
	v.Box.PackEnd(v.Container, true, true, 0)
}

func (v *View) Reset() {
	v.state.Reset()     // Reset the state variables.
	v.FaceView.Reset()  // Switch back to the main screen.
	v.InputView.Reset() // Reset the input.
	v.Container.Reset() // Clean all messages.

	// Recreate the message container if the type is different.
	if v.contType != msgIndex {
		v.createMessageContainer()
	}
}

// JoinServer is not thread-safe, but it calls backend functions asynchronously.
func (v *View) JoinServer(session cchat.Session, server ServerMessage, done func()) {
	// Reset before setting.
	v.Reset()

	// Set the screen to loading.
	v.FaceView.SetLoading()

	// Bind the state.
	v.state.bind(session, server)

	// Skipping ok check because sender can be nil. Without the empty
	// check, Go will panic.
	sender, _ := server.(cchat.ServerMessageSender)
	// We're setting this variable before actually calling JoinServer. This is
	// because new messages created by JoinServer will use this state for things
	// such as determinining if it's deletable or not.
	v.InputView.SetSender(session, sender)

	gts.Async(func() (func(), error) {
		// We can use a background context here, as the user can't go anywhere
		// that would require cancellation anyway. This is done in ui.go.
		s, err := server.JoinServer(context.Background(), v.Container)
		if err != nil {
			err = errors.Wrap(err, "Failed to join server")
			// Even if we're erroring out, we're running the done() callback
			// anyway.
			return func() { done(); v.SetError(err) }, err
		}

		return func() {
			// Run the done() callback.
			done()

			// Set the screen to the main one.
			v.FaceView.SetMain()

			// Set the cancel handler.
			v.state.setcurrent(s)

		}, nil
	})
}

func (v *View) AddPresendMessage(msg input.PresendMessage) func(error) {
	var presend = v.Container.AddPresendMessage(msg)

	return func(err error) {
		// Set the retry message.
		presend.SetSentError(err)
		// Only attach the menu once. Further retries do not need to be
		// reattached.
		presend.AttachMenu([]menu.Item{
			menu.SimpleItem("Retry", func() {
				presend.SetLoading()
				v.retryMessage(msg, presend)
			}),
		})
	}
}

// LatestMessageFrom returns the last message ID with that author.
func (v *View) LatestMessageFrom(userID string) (msgID string, ok bool) {
	return v.Container.LatestMessageFrom(userID)
}

// retryMessage sends the message.
func (v *View) retryMessage(msg input.PresendMessage, presend container.PresendGridMessage) {
	var sender = v.InputView.Sender
	if sender == nil {
		return
	}

	go func() {
		if err := sender.SendMessage(msg); err != nil {
			// Set the message's state to errored again, but we don't need to
			// rebind the menu.
			gts.ExecAsync(func() { presend.SetSentError(err) })
		}
	}()
}

// BindMenu attaches the menu constructor into the message with the needed
// states and callbacks.
func (v *View) BindMenu(msg container.GridMessage) {
	// Add 1 for the edit menu item.
	var mitems []menu.Item

	// Do we have editing capabilities? If yes, append a button to allow it.
	if v.InputView.Editable(msg.ID()) {
		mitems = append(mitems, menu.SimpleItem(
			"Edit", func() { v.InputView.StartEditing(msg.ID()) },
		))
	}

	// Do we have any custom actions? If yes, append it.
	if v.hasActions() {
		var actions = v.actioner.MessageActions(msg.ID())
		var items = make([]menu.Item, len(actions))

		for i, action := range actions {
			items[i] = v.makeActionItem(action, msg.ID())
		}

		mitems = append(mitems, items...)
	}

	msg.AttachMenu(mitems)
}

// makeActionItem creates a new menu callback that's called on menu item
// activation.
func (v *View) makeActionItem(action, msgID string) menu.Item {
	return menu.SimpleItem(action, func() {
		go func() {
			// Run, get the error, and try to log it. The logger will ignore nil
			// errors.
			err := v.state.actioner.DoMessageAction(action, msgID)
			log.Error(errors.Wrap(err, "Failed to do action "+action))
		}()
	})
}

// ServerMessage combines Server and ServerMessage from cchat.
type ServerMessage interface {
	cchat.Server
	cchat.ServerMessage
}

type state struct {
	session cchat.Session
	server  cchat.Server

	actioner cchat.ServerMessageActioner

	current func() // stop callback
	author  string
}

func (s *state) Reset() {
	// If we still have the last server to leave, then leave it.
	if s.current != nil {
		s.current()
	}

	// Lazy way to reset the state.
	*s = state{}
}

func (s *state) hasActions() bool {
	return s.actioner != nil
}

// SessionID returns the session ID, or an empty string if there's no session.
func (s *state) SessionID() string {
	if s.session != nil {
		return s.session.ID()
	}
	return ""
}

func (s *state) bind(session cchat.Session, server ServerMessage) {
	s.session = session
	s.server = server
	s.actioner, _ = server.(cchat.ServerMessageActioner)
}

func (s *state) setcurrent(fn func()) {
	s.current = fn
}
