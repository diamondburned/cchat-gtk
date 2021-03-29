package container

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

// BacklogLimit is the maximum number of messages to store in the container at
// once.
const BacklogLimit = 50

type MessageRow interface {
	message.Container
	GrabFocus()
}

type PresendMessageRow interface {
	MessageRow
	message.Presender
}

// Container is a generic messages container for children messages for children
// packages.
type Container interface {
	gtk.IWidget
	cchat.MessagesContainer

	// Reset resets the message container to its original state.
	Reset()

	// SetSelf sets the author for the current user.
	SetSelf(self *message.Author)

	// NewPresendMessage creates and adds a presend message state into the list.
	NewPresendMessage(state *message.PresendState) PresendMessageRow

	// AddMessage adds a new message into the list.
	AddMessage(row MessageRow)

	// FirstMessage returns the first message in the buffer. Nil is returned if
	// there's nothing.
	FirstMessage() MessageRow
	// LastMessage returns the last message in the buffer or nil if there's
	// nothing.
	LastMessage() MessageRow
	// Message finds and returns the message, if any. It performs maximum 2
	// constant-time lookups.
	Message(id cchat.ID, nonce string) MessageRow
	// FindMessage finds a message that satisfies the given callback. It
	// iterates the message buffer from latest to earliest.
	FindMessage(isMessage func(MessageRow) bool) MessageRow

	// Highlight temporarily highlights the given message for a short while.
	Highlight(msg MessageRow)

	// UI methods.

	SetFocusHAdjustment(*gtk.Adjustment)
	SetFocusVAdjustment(*gtk.Adjustment)
}

// UpdateMessage is a convenient function to update a message in the container.
// It does nothing if the message is not found.
func UpdateMessage(ct Container, update cchat.MessageUpdate) {
	if msg := ct.Message(update.ID(), ""); msg != nil {
		msg.UpdateContent(update.Content(), true)
	}
}

// LatestMessageFrom returns the latest message from the given author ID.
func LatestMessageFrom(ct Container, authorID cchat.ID) MessageRow {
	return ct.FindMessage(func(msg MessageRow) bool {
		return msg.Unwrap(false).Author.ID == authorID
	})
}

// Controller is for menu actions.
type Controller interface {
	// Connector is used for button press events to unselect messages.
	primitives.Connector
	// BindMenu expects the controller to add actioner into the message.
	BindMenu(MessageRow)
	// Bottomed returns whether or not the message scroller is at the bottom.
	Bottomed() bool
	// AuthorEvent is called on message create/update. This is used to update
	// the typer state.
	AuthorEvent(authorID cchat.ID)
	// SelectMessage is called when a message is selected.
	SelectMessage(list *ListStore, msg MessageRow)
	// UnselectMessage is called when the message selection is cleared.
	UnselectMessage()
}

const ColumnSpacing = 8

// ListContainer is an implementation of Container, which allows flexible
// message grids.
type ListContainer struct {
	*handy.Clamp

	*ListStore

	Controller
}

// messageRow w/ required internals
type messageRow struct {
	MessageRow
	presend message.Presender // this shouldn't be here but i'm lazy
}

// unwrapRow is a helper that unwraps a messageRow if it's not nil. If it's nil,
// then a nil interface is returned.
func unwrapRow(msg *messageRow) MessageRow {
	if msg == nil || msg.MessageRow == nil {
		return nil
	}
	return msg.MessageRow
}

func NewListContainer(ctrl Controller) *ListContainer {
	listStore := NewListStore(ctrl)
	listStore.ListBox.Show()

	clamp := handy.ClampNew()
	clamp.SetMaximumSize(800)
	clamp.SetTighteningThreshold(600)
	clamp.SetHExpand(true)
	clamp.SetVExpand(true)
	clamp.Add(listStore.ListBox)
	clamp.Show()

	return &ListContainer{
		Clamp:      clamp,
		ListStore:  listStore,
		Controller: ctrl,
	}
}

func (c *ListContainer) AddMessage(row MessageRow) {
	c.ListStore.AddMessage(row)
	c.CleanMessages()
}

// CleanMessages cleans up the oldest messages if the user is scrolled to the
// bottom. True is returned if there were changes.
func (c *ListContainer) CleanMessages() bool {
	// Determine if the user is scrolled to the bottom for cleaning up.
	if c.Bottomed() {
		// Clean up the backlog.
		if delta := c.MessagesLen() - BacklogLimit; delta > 0 {
			c.DeleteEarliest(delta)
			return true
		}
	}

	return false
}

func (c *ListContainer) SetFocusHAdjustment(adj *gtk.Adjustment) {
	c.ListBox.SetFocusHAdjustment(adj)
}
func (c *ListContainer) SetFocusVAdjustment(adj *gtk.Adjustment) {
	c.ListBox.SetFocusVAdjustment(adj)
}
