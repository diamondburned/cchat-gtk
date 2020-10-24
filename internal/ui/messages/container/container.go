package container

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/gotk3/gotk3/gtk"
)

// BacklogLimit is the maximum number of messages to store in the container at
// once.
const BacklogLimit = 35

type GridMessage interface {
	message.Container
	// Focusable should return a widget that can be focused.
	Focusable() gtk.IWidget
	// Attach should only be called once.
	Attach() []gtk.IWidget
	// AttachMenu should override the stored constructor.
	AttachMenu(items []menu.Item) // save memory
}

type PresendGridMessage interface {
	GridMessage
	message.PresendContainer
}

// Container is a generic messages container for children messages for children
// packages.
type Container interface {
	gtk.IWidget

	// Thread-safe methods.
	cchat.MessagesContainer

	// Thread-unsafe methods.
	CreateMessageUnsafe(cchat.MessageCreate)
	UpdateMessageUnsafe(cchat.MessageUpdate)
	DeleteMessageUnsafe(cchat.MessageDelete)

	// FirstMessage returns the first message in the buffer. Nil is returned if
	// there's nothing.
	FirstMessage() GridMessage
	// TranslateCoordinates is used for scrolling to the message.
	TranslateCoordinates(parent gtk.IWidget, msg GridMessage) (y int)
	// AddPresendMessage adds and displays an unsent message.
	AddPresendMessage(msg input.PresendMessage) PresendGridMessage
	// LatestMessageFrom returns the last message ID with that author.
	LatestMessageFrom(authorID string) (msgID string, ok bool)

	// UI methods.

	SetFocusHAdjustment(*gtk.Adjustment)
	SetFocusVAdjustment(*gtk.Adjustment)
}

// Controller is for menu actions.
type Controller interface {
	// BindMenu expects the controller to add actioner into the message.
	BindMenu(GridMessage)
	// Bottomed returns whether or not the message scroller is at the bottom.
	Bottomed() bool
	// AuthorEvent is called on message create/update. This is used to update
	// the typer state.
	AuthorEvent(a cchat.Author)
}

// Constructor is an interface for making custom message implementations which
// allows GridContainer to generically work with.
type Constructor interface {
	NewMessage(cchat.MessageCreate) GridMessage
	NewPresendMessage(input.PresendMessage) PresendGridMessage
}

const ColumnSpacing = 10

// GridContainer is an implementation of Container, which allows flexible
// message grids.
type GridContainer struct {
	*GridStore
	Controller
}

// gridMessage w/ required internals
type gridMessage struct {
	GridMessage
	presend message.PresendContainer // this shouldn't be here but i'm lazy
}

var _ Container = (*GridContainer)(nil)

func NewGridContainer(constr Constructor, ctrl Controller) *GridContainer {
	return &GridContainer{
		GridStore:  NewGridStore(constr, ctrl),
		Controller: ctrl,
	}
}

// CreateMessageUnsafe inserts a message as well as cleaning up the backlog if
// the user is scrolled to the bottom.
func (c *GridContainer) CreateMessageUnsafe(msg cchat.MessageCreate) {
	// Insert the message first.
	c.GridStore.CreateMessageUnsafe(msg)

	// Determine if the user is scrolled to the bottom for cleaning up.
	if c.Bottomed() {
		// Clean up the backlog.
		c.DeleteEarliest(c.MessagesLen() - BacklogLimit)
	}
}

func (c *GridContainer) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() { c.CreateMessageUnsafe(msg) })
}

func (c *GridContainer) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() { c.UpdateMessageUnsafe(msg) })
}

func (c *GridContainer) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() { c.DeleteMessageUnsafe(msg) })
}
