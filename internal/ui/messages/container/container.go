package container

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

// BacklogLimit is the maximum number of messages to store in the container at
// once.
const BacklogLimit = 50

type MessageRow interface {
	message.Container
	// Attach should only be called once.
	Row() *gtk.ListBoxRow
	// AttachMenu should override the stored constructor.
	AttachMenu(items []menu.Item) // save memory
	// MenuItems returns the list of attached menu items.
	MenuItems() []menu.Item
	// SetReferenceHighlighter sets the reference highlighter into the message.
	SetReferenceHighlighter(refer labeluri.ReferenceHighlighter)
}

type PresendMessageRow interface {
	MessageRow
	message.PresendContainer
}

// Container is a generic messages container for children messages for children
// packages.
type Container interface {
	gtk.IWidget

	// Reset resets the message container to its original state.
	Reset()

	// CreateMessageUnsafe creates a new message and returns the index that is
	// the location the message is added to.
	CreateMessageUnsafe(cchat.MessageCreate) MessageRow
	UpdateMessageUnsafe(cchat.MessageUpdate)
	DeleteMessageUnsafe(cchat.MessageDelete)

	// FirstMessage returns the first message in the buffer. Nil is returned if
	// there's nothing.
	FirstMessage() MessageRow
	// AddPresendMessage adds and displays an unsent message.
	AddPresendMessage(msg input.PresendMessage) PresendMessageRow
	// LatestMessageFrom returns the last message ID with that author.
	LatestMessageFrom(authorID string) (msgID string, ok bool)
	// Message finds and returns the message, if any.
	Message(id cchat.ID, nonce string) MessageRow
	// FindMessage finds a message that satisfies the given callback.
	FindMessage(isMessage func(MessageRow) bool) MessageRow

	// Highlight temporarily highlights the given message for a short while.
	Highlight(msg MessageRow)

	// UI methods.

	SetFocusHAdjustment(*gtk.Adjustment)
	SetFocusVAdjustment(*gtk.Adjustment)
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
	AuthorEvent(a cchat.Author)
	// SelectMessage is called when a message is selected.
	SelectMessage(list *ListStore, msg MessageRow)
	// UnselectMessage is called when the message selection is cleared.
	UnselectMessage()
}

// Constructor is an interface for making custom message implementations which
// allows ListContainer to generically work with.
type Constructor struct {
	NewMessage        func(msg cchat.MessageCreate, before MessageRow) MessageRow
	NewPresendMessage func(msg input.PresendMessage, before MessageRow) PresendMessageRow
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
	presend message.PresendContainer // this shouldn't be here but i'm lazy
}

// unwrapRow is a helper that unwraps a messageRow if it's not nil. If it's nil,
// then a nil interface is returned.
func unwrapRow(msg *messageRow) MessageRow {
	if msg == nil || msg.MessageRow == nil {
		return nil
	}
	return msg.MessageRow
}

var _ Container = (*ListContainer)(nil)

func NewListContainer(ctrl Controller, constr Constructor) *ListContainer {
	listStore := NewListStore(ctrl, constr)
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

// TODO: remove useless abstraction (this file).

// // CreateMessageUnsafe inserts a message. It does not clean up old messages.
// func (c *ListContainer) CreateMessageUnsafe(msg cchat.MessageCreate) MessageRow {
// 	return c.ListStore.CreateMessageUnsafe(msg)
// }

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
