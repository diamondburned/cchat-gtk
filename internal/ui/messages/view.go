package messages

import (
	"context"
	"runtime"
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
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/sadface"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/typing"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/drag"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
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

type MessagesContainer interface {
	gtk.IWidget
	cchat.MessagesContainer
	container.Container
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
	Container MessagesContainer
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
	view := &View{
		ctrl:     c,
		contType: -1, // force recreate
	}

	view.Typing = typing.New()
	view.Typing.Show()

	view.MemberList = memberlist.New(view)
	view.MemberList.Show()

	view.MsgBox, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	view.MsgBox.PackEnd(view.Typing, false, false, 0)
	view.MsgBox.Show()

	view.Scroller = autoscroll.NewScrolledWindow()
	view.Scroller.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	view.Scroller.SetVExpand(true)
	view.Scroller.SetHExpand(true)
	view.Scroller.Add(view.MsgBox)
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

	view.InputView = input.NewView(view, view.Typing)
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
	logo, _ := gtk.ImageNew()
	logo.SetFromSurface(icons.Logo256Variant2(128, logo.GetScaleFactor()))
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
	// If we still want the same type of message container, then we don't need
	// to remake a new one.
	if v.contType == msgIndex {
		v.Container.Reset()
		return
	}

	// Remove the old message container.
	if v.Container != nil {
		v.Container.Reset()
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

// Reset resets the message view.
func (v *View) Reset() {
	v.FaceView.Reset() // Switch back to the main screen.
	v.reset()
}

// reset resets the message view, but does not change visible containers.
func (v *View) reset() {
	v.state.Reset()      // Reset the state variables.
	v.Header.Reset()     // Reset the header.
	v.Typing.Reset()     // Reset the typing state.
	v.InputView.Reset()  // Reset the input.
	v.MemberList.Reset() // Reset the member list.

	// Bring the leaflet view back to the message.
	v.Leaflet.SetVisibleChild(v.LeftBox)

	// Keep the scroller at the bottom.
	v.Scroller.Bottomed = true

	// Reallocate the entire message container.
	v.createMessageContainer()
}

func (v *View) SetFolded(folded bool) {
	v.parentFolded = folded
	v.Header.SetMiniBreadcrumb(folded)
	v.Header.MessageCtrl.SetHidden(folded)
	v.InputView.Username.SetRevealChild(!folded)

	// Hide the member list automatically on folded.
	if folded {
		v.Header.ShowMembers.SetActive(false)
	}
}

// MemberListUpdated is called everytime the member list is updated.
func (v *View) MemberListUpdated(c *memberlist.Container) {
	// We can show the members list if it's not empty.
	empty := c.IsEmpty()
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
func (v *View) JoinServer(ses *session.Row, srv *server.ServerRow, bc traverse.Breadcrumber) {
	// Set the screen to loading.
	v.FaceView.SetLoading()
	v.ctrl.OnMessageBusy()

	// Reset before setting.
	v.reset()

	// Get the messenger once.
	var messenger = srv.Server.AsMessenger()
	// Exit if this server is not a messenger.
	if messenger == nil {
		return
	}

	// Bind the state.
	v.state.bind(ses.Session, srv.Server, messenger)

	// We're setting this variable before actually calling JoinServer. This is
	// because new messages created by JoinServer will use this state for things
	// such as determinining if it's deletable or not.
	v.InputView.SetMessenger(ses.Session, messenger)

	// Bind the container's self user to what we just set.
	v.Container.SetSelf(v.InputView.Username.State)

	go func() {
		// We can use a background context here, as the user can't go anywhere
		// that would require cancellation anyway. This is done in ui.go.
		s, err := messenger.JoinServer(context.Background(), v.Container)
		if err != nil {
			log.Error(errors.Wrap(err, "Failed to join server"))
			// Even if we're erroring out, we're running the done() callback
			// anyway.
			gts.ExecAsync(func() {
				v.ctrl.OnMessageDone()
				v.FaceView.SetError(err)
			})
			return
		}

		gts.ExecAsync(func() {
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
		})

		// Collect garbage after a channel switch since a lot of images will
		// need to be freed.
		runtime.GC()
	}()
}

func (v *View) FetchBacklog() {
	backlogger := v.state.Backlogger()
	if backlogger == nil {
		return
	}

	firstMsg := container.FirstMessage(v.Container)
	if firstMsg == nil {
		return
	}

	// Set the window as busy. TODO: loading circles.
	v.ctrl.OnMessageBusy()

	done := func() {
		v.ctrl.OnMessageDone()
		v.Container.Highlight(firstMsg)
	}

	firstID := firstMsg.Unwrap().ID

	gts.Async(func() (func(), error) {
		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()

		err := backlogger.Backlog(ctx, firstID, v.Container)
		return done, errors.Wrap(err, "Failed to get messages before ID")
	})
}

// AuthorEvent should be called on message create/update/delete.
func (v *View) AuthorEvent(authorID cchat.ID) {
	// Remove the author from the typing list if it's not nil.
	if authorID != "" {
		v.Typing.RemoveAuthor(authorID)
	}
}

// MessageAuthor returns the author from the message with the given ID.
func (v *View) MessageAuthor(msgID cchat.ID) *message.Author {
	msg := v.Container.Message(msgID, "")
	if msg == nil {
		return nil
	}

	return msg.Unwrap().Author
}

// Author returns the author from the message list with the given author ID.
func (v *View) Author(authorID cchat.ID) rich.LabelStateStorer {
	msg, _ := v.Container.FindMessage(func(msg container.MessageRow) bool {
		return msg.Unwrap().Author.ID == authorID
	})
	if msg == nil {
		return nil
	}

	state := msg.Unwrap()
	return &state.Author.Name
}

// LatestMessageFrom returns the last message ID with that author.
func (v *View) LatestMessageFrom(userID cchat.ID) container.MessageRow {
	msg, _ := container.LatestMessageFrom(v.Container, userID)
	return msg
}

func (v *View) SendMessage(msg message.PresendMessage) {
	state := message.NewPresendState(v.InputView.Username.State, msg)
	msgr := v.Container.NewPresendMessage(state)
	v.retryMessage(state, msgr)
}

// retryMessage sends the message.
func (v *View) retryMessage(state *message.PresendState, presend container.PresendMessageRow) {
	var sender = v.InputView.Sender
	if sender == nil {
		return
	}

	// Ensure the message is set to loading.
	presend.SetLoading()

	go func() {
		err := sender.Send(presend.SendingMessage())
		if err == nil {
			return
		}

		// Set the message's state to errored again, but we don't need to rebind
		// the menu.
		gts.ExecAsync(func() {
			presend.SetSentError(err)

			state.MenuItems = []menu.Item{
				menu.SimpleItem("Retry", func() {
					presend.SetLoading()
					v.retryMessage(state, presend)
				}),
			}
		})
	}()
}

var messageItemNames = MessageItemNames{
	Reply:  "Reply",
	Edit:   "Edit",
	Delete: "Delete",
}

// BindMenu attaches the menu constructor into the message with the needed
// states and callbacks.
func (v *View) BindMenu(msg container.MessageRow) {
	state := msg.Unwrap()

	// Add 1 for the edit menu item.
	var mitems = []menu.Item{
		menu.SimpleItem(
			"Reply", func() { v.InputView.StartReplyingTo(state.ID) },
		),
	}

	// Do we have editing capabilities? If yes, append a button to allow it.
	if v.InputView.Editable(state.ID) {
		mitems = append(mitems, menu.SimpleItem(
			"Edit", func() { v.InputView.StartEditing(state.ID) },
		))
	}

	// Do we have any custom actions? If yes, append it.
	if v.hasActions() {
		var actions = v.actioner.Actions(state.ID)
		var items = make([]menu.Item, len(actions))

		for i, action := range actions {
			items[i] = v.makeActionItem(action, state.ID)
		}

		mitems = append(mitems, items...)
	}

	state.MenuItems = mitems
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

// SelectMessage is called when a message is selected.
func (v *View) SelectMessage(_ *container.ListStore, msg container.MessageRow) {
	// Hijack the message's action list to search for what we have above.
	v.Header.MessageCtrl.Enable(msg, messageItemNames)
}

// UnselectMessage is called when the message selection is cleared.
func (v *View) UnselectMessage() {
	v.Header.MessageCtrl.Disable()
}
