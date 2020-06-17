package messages

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container/cozy"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/sadface"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// ServerMessage combines Server and ServerMessage from cchat.
type ServerMessage interface {
	cchat.Server
	cchat.ServerMessage
}

type state struct {
	session cchat.Session
	server  cchat.Server

	actioner cchat.ServerMessageActioner
	actions  []string

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
	return s.actioner != nil && len(s.actions) > 0
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
	if s.actioner, _ = server.(cchat.ServerMessageActioner); s.actioner != nil {
		s.actions = s.actioner.MessageActions()
	}
}

func (s *state) setcurrent(fn func()) {
	s.current = fn
}

type View struct {
	*sadface.FaceView
	Box *gtk.Box

	InputView *input.InputView
	Container container.Container

	// Inherit some useful methods.
	state
}

func NewView() *View {
	view := &View{}

	// TODO: change
	view.InputView = input.NewView(view)
	// view.Container = compact.NewContainer(view)
	view.Container = cozy.NewContainer(view)

	view.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	view.Box.PackStart(view.Container, true, true, 0)
	view.Box.PackStart(view.InputView, false, false, 0)
	view.Box.Show()

	// placeholder logo
	logo, _ := gtk.ImageNewFromPixbuf(icons.Logo256())
	logo.Show()

	view.FaceView = sadface.New(view.Box, logo)
	return view
}

func (v *View) Reset() {
	v.state.Reset()     // Reset the state variables.
	v.FaceView.Reset()  // Switch back to the main screen.
	v.Container.Reset() // Clean all messages.
	v.InputView.Reset() // Reset the input.
}

// JoinServer is not thread-safe, but it calls backend functions asynchronously.
func (v *View) JoinServer(session cchat.Session, server ServerMessage, done func()) {
	// Reset before setting.
	v.Reset()

	// Set the screen to loading.
	v.FaceView.SetLoading()

	// Bind the state.
	v.state.bind(session, server)

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

			// Skipping ok check because sender can be nil. Without the empty
			// check, Go will panic.
			sender, _ := server.(cchat.ServerMessageSender)
			v.InputView.SetSender(session, sender)
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
		presend.AttachMenu(func() []gtk.IMenuItem {
			return []gtk.IMenuItem{
				primitives.MenuItem("Retry", func() {
					presend.SetLoading()
					v.retryMessage(msg, presend)
				}),
			}
		})
	}
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
	// Don't bind anything if we don't have anything.
	if !v.state.hasActions() {
		return
	}

	msg.AttachMenu(func() []gtk.IMenuItem {
		var mitems = make([]gtk.IMenuItem, len(v.state.actions))
		for i, action := range v.state.actions {
			mitems[i] = primitives.MenuItem(action, v.menuItemActivate(msg.ID()))
		}
		return mitems
	})
}

// menuItemActivate creates a new callback that's called on menu item
// activation.
func (v *View) menuItemActivate(msgID string) func(m *gtk.MenuItem) {
	return func(m *gtk.MenuItem) {
		go func(action string) {
			// Run, get the error, and try to log it. The logger will ignore nil
			// errors.
			err := v.state.actioner.DoMessageAction(action, msgID)
			log.Error(errors.Wrap(err, "Failed to do action "+action))
		}(m.GetLabel())
	}
}
