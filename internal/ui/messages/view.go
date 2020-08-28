package messages

import (
	"context"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/icons"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container/compact"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container/cozy"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/memberlist"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/sadface"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/typing"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/drag"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
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

type Controller interface {
	// OnMessageBusy is called when the message buffer is busy. This happens
	// when it's loading messages.
	OnMessageBusy()
	// OnMessageDone is called after OnMessageBusy, when the message buffer is
	// done with loading.
	OnMessageDone()
}

type View struct {
	*gtk.Box

	Header *Header

	FaceView *sadface.FaceView
	Grid     *gtk.Grid

	Scroller  *autoscroll.ScrolledWindow
	InputView *input.InputView

	MsgBox    *gtk.Box
	Typing    *typing.Container
	Container container.Container
	contType  int // msgIndex

	MemberList *memberlist.Container

	// Inherit some useful methods.
	state

	ctrl Controller
}

func NewView(c Controller) *View {
	view := &View{ctrl: c}
	view.Typing = typing.New()
	view.Typing.Show()

	view.MemberList = memberlist.New()
	view.MemberList.Show()

	view.MsgBox, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	view.MsgBox.PackEnd(view.Typing, false, false, 0)
	view.MsgBox.Show()

	view.Scroller = autoscroll.NewScrolledWindow()
	view.Scroller.Add(view.MsgBox)
	view.Scroller.SetVExpand(true)
	view.Scroller.SetHExpand(true)
	view.Scroller.Show()

	view.MsgBox.SetFocusHAdjustment(view.Scroller.GetHAdjustment())
	view.MsgBox.SetFocusVAdjustment(view.Scroller.GetVAdjustment())

	// Create the message container, which will use PackEnd to add the widget on
	// TOP of the typing indicator.
	view.createMessageContainer()

	// Fetch the message backlog when the user has scrolled to the top.
	view.Scroller.Connect("edge-reached", func(_ *gtk.ScrolledWindow, p gtk.PositionType) {
		if p == gtk.POS_TOP {
			view.FetchBacklog()
		}
	})

	// A separator to go inbetween.
	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	sep.SetHExpand(true)
	sep.Show()

	view.InputView = input.NewView(view)
	view.InputView.SetHExpand(true)
	view.InputView.Show()

	view.Grid, _ = gtk.GridNew()
	view.Grid.Attach(view.Scroller, 0, 0, 1, 1)
	view.Grid.Attach(sep, 0, 1, 1, 1)
	view.Grid.Attach(view.InputView, 0, 2, 1, 1)
	view.Grid.Attach(view.MemberList, 1, 0, 1, 3)
	view.Grid.Show()

	primitives.AddClass(view.Grid, "message-view")

	// Bind a file drag-and-drop box into the main view box.
	drag.BindFileDest(view.Grid, view.InputView.Attachments.AddFiles)

	// placeholder logo
	logo, _ := gtk.ImageNewFromPixbuf(icons.Logo256Variant2(128))
	logo.Show()

	view.FaceView = sadface.New(view.Grid, logo)
	view.FaceView.Show()

	view.Header = NewHeader()
	view.Header.Show()

	view.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	view.Box.PackStart(view.Header, false, false, 0)
	view.Box.PackStart(view.FaceView, true, true, 0)

	return view
}

func (v *View) createMessageContainer() {
	// Remove the old message container.
	if v.Container != nil {
		v.MsgBox.Remove(v.Container)
	}

	// Update the container type.
	switch v.contType = msgIndex; msgIndex {
	case cozyMessage:
		v.Container = cozy.NewContainer(v)
	case compactMessage:
		v.Container = compact.NewContainer(v)
	}

	v.Container.SetFocusHAdjustment(v.Scroller.GetHAdjustment())
	v.Container.SetFocusVAdjustment(v.Scroller.GetVAdjustment())

	// Add the new message container.
	v.MsgBox.PackEnd(v.Container, true, true, 0)
}

func (v *View) Bottomed() bool { return v.Scroller.Bottomed }

func (v *View) Reset() {
	v.Header.Reset()     // Reset the header.
	v.state.Reset()      // Reset the state variables.
	v.Typing.Reset()     // Reset the typing state.
	v.InputView.Reset()  // Reset the input.
	v.MemberList.Reset() // Reset the member list.
	v.FaceView.Reset()   // Switch back to the main screen.

	// Keep the scroller at the bottom.
	v.Scroller.Bottomed = true

	// Reallocate the entire message container.
	v.createMessageContainer()
}

// JoinServer is not thread-safe, but it calls backend functions asynchronously.
func (v *View) JoinServer(session cchat.Session, server ServerMessage) {
	// Reset before setting.
	v.Reset()

	// Set the screen to loading.
	v.FaceView.SetLoading()
	v.ctrl.OnMessageBusy()

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
			return func() { v.ctrl.OnMessageDone(); v.FaceView.SetError(err) }, err
		}

		return func() {
			// Run the done() callback.
			v.ctrl.OnMessageDone()

			// Set the screen to the main one.
			v.FaceView.SetMain()

			// Set the cancel handler.
			v.state.setcurrent(s)

			// Try setting the typing indicator if available.
			v.Typing.TrySubscribe(server)

			// Try and use the list.
			v.MemberList.TryAsyncList(server)
		}, nil
	})
}

func (v *View) FetchBacklog() {
	var backlogger = v.state.Backlogger()
	if backlogger == nil {
		return
	}

	var firstMsg = v.Container.FirstMessage()
	if firstMsg == nil {
		return
	}

	// Set the window as busy. TODO: loading circles.
	v.ctrl.OnMessageBusy()

	var done = func() {
		v.ctrl.OnMessageDone()

		// Restore scrolling.
		y := v.Container.TranslateCoordinates(v.MsgBox, firstMsg)
		v.Scroller.GetVAdjustment().SetValue(float64(y))
	}

	gts.Async(func() (func(), error) {
		err := backlogger.MessagesBefore(context.Background(), firstMsg.ID(), v.Container)
		return done, errors.Wrap(err, "Failed to get messages before ID")
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

// AuthorEvent should be called on message create/update/delete.
func (v *View) AuthorEvent(author cchat.MessageAuthor) {
	// Remove the author from the typing list if it's not nil.
	if author != nil {
		v.Typing.RemoveAuthor(author)
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

	actioner   cchat.ServerMessageActioner
	backlogger cchat.ServerMessageBacklogger

	current func() // stop callback
	author  string

	lastBacklogged time.Time
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

const backloggingFreq = time.Second * 3

// Backlogger returns the backlogger instance if it's allowed to fetch more
// backlogs.
func (s *state) Backlogger() cchat.ServerMessageBacklogger {
	if s.backlogger == nil || s.current == nil {
		return nil
	}

	var now = time.Now()

	if s.lastBacklogged.Add(backloggingFreq).After(now) {
		return nil
	}

	s.lastBacklogged = now
	return s.backlogger
}

func (s *state) bind(session cchat.Session, server ServerMessage) {
	s.session = session
	s.server = server
	s.actioner, _ = server.(cchat.ServerMessageActioner)
	s.backlogger, _ = server.(cchat.ServerMessageBacklogger)
}

func (s *state) setcurrent(fn func()) {
	s.current = fn
}
