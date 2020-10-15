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
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/handy"
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
	// GoBack tells the main leaflet to go back to the services list.
	GoBack()
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
	Leaflet  *handy.Leaflet

	LeftBox   *gtk.Box
	Scroller  *autoscroll.ScrolledWindow
	InputView *input.InputView

	MsgBox    *gtk.Box
	Typing    *typing.Container
	Container container.Container
	contType  int // msgIndex

	MemberList *memberlist.Container // right box

	// Inherit some useful methods.
	state

	ctrl         Controller
	parentFolded bool // folded state
}

var messageStack = primitives.PrepareClassCSS("message-stack", `
	.message-stack {
		background-color: mix(@theme_bg_color, @theme_fg_color, 0.03);
	}
`)

var messageScroller = primitives.PrepareClassCSS("message-scroller", ``)

func NewView(c Controller) *View {
	view := &View{ctrl: c}
	view.Typing = typing.New()
	view.Typing.Show()

	view.MemberList = memberlist.New(view)
	view.MemberList.Show()

	view.MsgBox, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	view.MsgBox.PackEnd(view.Typing, false, false, 0)
	view.MsgBox.Show()

	view.Scroller = autoscroll.NewScrolledWindow()
	view.Scroller.Add(view.MsgBox)
	view.Scroller.SetVExpand(true)
	view.Scroller.SetHExpand(true)
	view.Scroller.Show()
	messageScroller(view.Scroller)

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

	view.LeftBox, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	view.LeftBox.PackStart(view.Scroller, true, true, 0)
	view.LeftBox.PackStart(sep, false, false, 0)
	view.LeftBox.PackStart(view.InputView, false, false, 0)
	view.LeftBox.Show()

	view.Leaflet = handy.LeafletNew()
	view.Leaflet.Add(view.LeftBox)
	view.Leaflet.Add(view.MemberList)
	view.Leaflet.SetVisibleChild(view.LeftBox)
	view.Leaflet.Show()
	primitives.AddClass(view.Leaflet, "message-view")

	// Bind a file drag-and-drop box into the main view box.
	drag.BindFileDest(view.LeftBox, view.InputView.Attachments.AddFiles)

	// placeholder logo
	logo, _ := gtk.ImageNewFromPixbuf(icons.Logo256Variant2(128))
	logo.Show()

	view.FaceView = sadface.New(view.Leaflet, logo)
	view.FaceView.Show()
	messageStack(view.FaceView)

	view.Header = NewHeader()
	view.Header.Show()
	view.Header.OnBackPressed(view.ctrl.GoBack)
	view.Header.OnShowMembersToggle(func(show bool) {
		// If the leaflet is folded, then we should always reveal the child. Its
		// visibility should be determined by the leaflet's state.
		if view.parentFolded {
			view.MemberList.SetRevealChild(true)
			if show {
				view.Leaflet.SetVisibleChild(view.MemberList)
			} else {
				view.Leaflet.SetVisibleChild(view.LeftBox)
			}
		} else {
			// Leaflet's visible child does not matter if it's not folded,
			// though we should still set the visible child to LeftBox in case
			// that changes.
			view.MemberList.SetRevealChild(show)
			view.Leaflet.SetVisibleChild(view.LeftBox)
		}
	})

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

	// Bring the leaflet view back to the message.
	v.Leaflet.SetVisibleChild(v.LeftBox)

	// Keep the scroller at the bottom.
	v.Scroller.Bottomed = true

	// Reallocate the entire message container.
	v.createMessageContainer()
}

func (v *View) SetFolded(folded bool) {
	v.parentFolded = folded

	// Change to a mini breadcrumb if we're collapsed.
	v.Header.SetMiniBreadcrumb(folded)

	// Show the right back button if we're collapsed.
	v.Header.SetShowBackButton(folded)

	// Hide the username in the input bar if we're collapsed.
	v.InputView.Username.SetRevealChild(!folded)

	// Hide the member list automatically on folded.
	if folded {
		v.Header.ShowMembers.SetActive(false)
	}
}

// MemberListUpdated is called everytime the member list is updated.
func (v *View) MemberListUpdated(c *memberlist.Container) {
	// We can show the members list if it's not empty.
	var empty = c.IsEmpty()
	v.Header.SetCanShowMembers(!empty)

	// If the member list is now empty, then hide the entire thing.
	if empty {
		// We can set active to false, which would trigger the above callback
		// and hide the member list.
		v.Header.ShowMembers.SetActive(false)
	} else {
		// Restore visibility.
		if !v.Leaflet.GetFolded() && v.Header.ShowMembers.GetActive() {
			c.SetRevealChild(true)
		}
	}
}

// JoinServer is not thread-safe, but it calls backend functions asynchronously.
func (v *View) JoinServer(session cchat.Session, server cchat.Server, bc traverse.Breadcrumber) {
	// Reset before setting.
	v.Reset()

	// Set the screen to loading.
	v.FaceView.SetLoading()
	v.ctrl.OnMessageBusy()

	// Get the messenger once.
	var messenger = server.AsMessenger()
	// Exit if this server is not a messenger.
	if messenger == nil {
		return
	}

	// Bind the state.
	v.state.bind(session, server, messenger)

	// We're setting this variable before actually calling JoinServer. This is
	// because new messages created by JoinServer will use this state for things
	// such as determinining if it's deletable or not.
	v.InputView.SetMessenger(session, messenger)

	gts.Async(func() (func(), error) {
		// We can use a background context here, as the user can't go anywhere
		// that would require cancellation anyway. This is done in ui.go.
		s, err := messenger.JoinServer(context.Background(), v.Container)
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

			// Set the headerbar's breadcrumb.
			v.Header.SetBreadcrumber(bc)

			// Try setting the typing indicator if available.
			v.Typing.TrySubscribe(messenger)

			// Try and use the list.
			v.MemberList.TryAsyncList(messenger)
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
		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()

		err := backlogger.Backlog(ctx, firstMsg.ID(), v.Container)
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
func (v *View) AuthorEvent(author cchat.Author) {
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
		if err := sender.Send(msg); err != nil {
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
		var actions = v.actioner.Actions(msg.ID())
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
			err := v.state.actioner.Do(action, msgID)
			log.Error(errors.Wrap(err, "Failed to do action "+action))
		}()
	})
}

// ServerMessage combines Server and ServerMessage from cchat.
type ServerMessage interface {
	cchat.Server
	cchat.Messenger
}

type state struct {
	session cchat.Session
	server  cchat.Server

	actioner   cchat.Actioner
	backlogger cchat.Backlogger

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

// ServerID returns the server ID, or an empty string if there's no server.
func (s *state) ServerID() string {
	if s.server != nil {
		return s.server.ID()
	}
	return ""
}

const backloggingFreq = time.Second * 3

// Backlogger returns the backlogger instance if it's allowed to fetch more
// backlogs.
func (s *state) Backlogger() cchat.Backlogger {
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

func (s *state) bind(session cchat.Session, server cchat.Server, msgr cchat.Messenger) {
	s.session = session
	s.server = server
	s.actioner = msgr.AsActioner()
	s.backlogger = msgr.AsBacklogger()
}

func (s *state) setcurrent(fn func()) {
	s.current = fn
}
