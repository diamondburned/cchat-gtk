package container

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/gotk3/gotk3/gtk"
)

type GridMessage interface {
	message.Container
	// Attach should only be called once.
	Attach(grid *gtk.Grid, row int)
	// AttachMenu should override the stored constructor.
	AttachMenu(constructor func() []gtk.IMenuItem) // save memory
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
	ScrollToBottom()

	// AddPresendMessage adds and displays an unsent message.
	AddPresendMessage(msg input.PresendMessage) PresendGridMessage
}

// Controller is for menu actions.
type Controller interface {
	// BindMenu expects the controller to add actioner into the message.
	BindMenu(GridMessage)
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
	*autoscroll.ScrolledWindow
	*GridStore
}

// gridMessage w/ required internals
type gridMessage struct {
	GridMessage
	presend message.PresendContainer // this shouldn't be here but i'm lazy
}

var _ Container = (*GridContainer)(nil)

func NewGridContainer(constr Constructor, ctrl Controller) *GridContainer {
	store := NewGridStore(constr, ctrl)

	sw := autoscroll.NewScrolledWindow()
	sw.Add(store.Grid)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)
	sw.Show()

	return &GridContainer{
		ScrolledWindow: sw,
		GridStore:      store,
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

// Reset is not thread-safe.
func (c *GridContainer) Reset() {
	c.GridStore.Reset()
	c.ScrolledWindow.Bottomed = true
}
