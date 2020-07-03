package container

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/gotk3/gotk3/gtk"
)

// BacklogLimit is the maximum number of messages to store in the container at
// once.
const BacklogLimit = 35

type GridMessage interface {
	message.Container
	// Attach should only be called once.
	Attach(grid *gtk.Grid, row int)
	// AttachMenu should override the stored constructor.
	AttachMenu(items []menu.Item) // save memory
}

func AttachRow(grid *gtk.Grid, row int, widgets ...gtk.IWidget) {
	for i, w := range widgets {
		grid.Attach(w, i, row, 1, 1)
	}
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

	Reset()

	// AddPresendMessage adds and displays an unsent message.
	AddPresendMessage(msg input.PresendMessage) PresendGridMessage
	// LatestMessageFrom returns the last message ID with that author.
	LatestMessageFrom(authorID string) (msgID string, ok bool)
}

// Controller is for menu actions.
type Controller interface {
	// BindMenu expects the controller to add actioner into the message.
	BindMenu(GridMessage)
	// Bottomed returns whether or not the message scroller is at the bottom.
	Bottomed() bool
	// ScrollToBottom scrolls the message view to the bottom.
	// ScrollToBottom()
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
	if !c.Bottomed() {
		return
	}

	// Clean up the backlog.
	if clean := len(c.messages) - BacklogLimit; clean > 0 {
		// Remove them from the map and the container.
		for _, id := range c.messageIDs[:clean] {
			delete(c.messages, id)
			// We can gradually pop the first item off here, as we're removing
			// from 0th, and items are being shifted backwards.
			c.Grid.RemoveRow(0)
		}

		// Cut the message IDs away by shifting the slice.
		c.messageIDs = append(c.messageIDs[:0], c.messageIDs[clean:]...)
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
