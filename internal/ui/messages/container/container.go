package container

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type GridMessage interface {
	message.Container
	Attach(grid *gtk.Grid, row int)
}

type PresendGridMessage interface {
	GridMessage
	message.PresendContainer
}

// Constructor is an interface for making custom message implementations which
// allows GridContainer to generically work with.
type Constructor interface {
	NewMessage(cchat.MessageCreate) GridMessage
	NewPresendMessage(input.PresendMessage) PresendGridMessage
}

// Container is a generic messages container.
type Container interface {
	gtk.IWidget
	cchat.MessagesContainer

	Reset()
	ScrollToBottom()

	// PresendMessage is for unsent messages.
	PresendMessage(input.PresendMessage) (done func(sendError error))
}

func AttachRow(grid *gtk.Grid, row int, widgets ...gtk.IWidget) {
	for i, w := range widgets {
		grid.Attach(w, i, row, 1, 1)
	}
}

const ColumnSpacing = 10

// GridContainer is an implementation of Container, which allows flexible
// message grids.
type GridContainer struct {
	*autoscroll.ScrolledWindow
	Main *gtk.Grid

	construct Constructor

	messages   []*gridMessage // sync w/ grid rows
	messageIDs map[string]int
	nonceMsgs  map[string]int
}

// gridMessage w/ required internals
type gridMessage struct {
	GridMessage
	presend message.PresendContainer // this shouldn't be here but i'm lazy
}

var (
	_ Container               = (*GridContainer)(nil)
	_ cchat.MessagesContainer = (*GridContainer)(nil)
)

func NewGridContainer(constr Constructor) *GridContainer {
	grid, _ := gtk.GridNew()
	grid.SetColumnSpacing(ColumnSpacing)
	grid.SetRowSpacing(5)
	grid.SetMarginStart(5)
	grid.SetMarginEnd(5)
	grid.SetMarginBottom(5)
	grid.Show()

	sw := autoscroll.NewScrolledWindow()
	sw.Add(grid)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)
	sw.Show()

	container := GridContainer{
		ScrolledWindow: sw,
		Main:           grid,
		construct:      constr,
		messageIDs:     map[string]int{},
		nonceMsgs:      map[string]int{},
	}

	return &container
}

func (c *GridContainer) Reset() {
	c.Main.GetChildren().Foreach(func(v interface{}) {
		// Unsafe assertion ftw.
		c.Main.Remove(v.(gtk.IWidget))
	})

	c.messages = nil
	c.messageIDs = map[string]int{}
	c.nonceMsgs = map[string]int{}

	c.ScrolledWindow.Bottomed = true
}

// PresendMessage is not thread-safe.
func (c *GridContainer) PresendMessage(msg input.PresendMessage) func(error) {
	presend := c.construct.NewPresendMessage(msg)

	msgc := &gridMessage{
		GridMessage: presend,
		presend:     presend,
	}

	// Grab index before appending, as that'll be where the added message is.
	index := len(c.messages)

	c.messages = append(c.messages, msgc)

	c.nonceMsgs[presend.Nonce()] = index
	msgc.Attach(c.Main, index)

	return func(err error) {
		if err != nil {
			presend.SetSentError(err)
			log.Error(errors.Wrap(err, "Failed to send message"))
		}
	}
}

// FindMessage iterates backwards and returns the message if isMessage() returns
// true on that message.
func (c *GridContainer) FindMessage(isMessage func(msg GridMessage) bool) GridMessage {
	for i := len(c.messages) - 1; i >= 0; i-- {
		if msg := c.messages[i].GridMessage; isMessage(msg) {
			return msg
		}
	}
	return nil
}

// Message finds the message state in the container. It is not thread-safe. This
// exists for backwards compatibility.
func (c *GridContainer) Message(msg cchat.MessageHeader) GridMessage {
	if m := c.message(msg); m != nil {
		return m.GridMessage
	}
	return nil
}

func (c *GridContainer) message(msg cchat.MessageHeader) *gridMessage {
	// Search using the ID first.
	i, ok := c.messageIDs[msg.ID()]
	if ok {
		return c.messages[i]
	}

	// Is this an existing message?
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		var nonce = noncer.Nonce()

		// Things in this map are guaranteed to have presend != nil.
		i, ok := c.nonceMsgs[nonce]
		if ok {
			// Move the message outside nonceMsgs and into messageIDs.
			delete(c.nonceMsgs, nonce)
			c.messageIDs[msg.ID()] = i

			// Get the message pointer.
			m := c.messages[i]

			// Set the right ID.
			m.presend.SetID(msg.ID())
			m.presend.SetDone()
			// Destroy the presend struct.
			m.presend = nil

			return m
		}
	}

	return nil
}

func (c *GridContainer) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		// Attempt to update before insertion (aka upsert).
		if msgc := c.Message(msg); msgc != nil {
			msgc.UpdateAuthor(msg.Author())
			msgc.UpdateContent(msg.Content())
			msgc.UpdateTimestamp(msg.Time())
			return
		}

		msgc := &gridMessage{
			GridMessage: c.construct.NewMessage(msg),
		}

		// Grab index before appending, as that'll be where the added message is.
		index := len(c.messages)

		c.messages = append(c.messages, msgc)

		c.messageIDs[msgc.ID()] = index
		msgc.Attach(c.Main, index)
	})
}

func (c *GridContainer) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() {
		if msgc := c.Message(msg); msgc != nil {
			if author := msg.Author(); author != nil {
				msgc.UpdateAuthor(author)
			}
			if content := msg.Content(); !content.Empty() {
				msgc.UpdateContent(content)
			}
		}
	})
}

func (c *GridContainer) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() {
		// TODO: add nonce check.
		if i, ok := c.messageIDs[msg.ID()]; ok {
			// Remove off the slice.
			c.messages = append(c.messages[:i], c.messages[i+1:]...)

			// Remove off the map.
			delete(c.messageIDs, msg.ID())
			c.Main.RemoveRow(i)
		}
	})
}
