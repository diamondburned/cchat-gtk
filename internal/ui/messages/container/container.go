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

// gridMessage w/ required internals
type gridMessage struct {
	GridMessage
	presend message.PresendContainer // this shouldn't be here but i'm lazy
	index   int
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

// GridContainer is an implementation of Container, which allows flexible
// message grids.
type GridContainer struct {
	*autoscroll.ScrolledWindow
	Main *gtk.Grid

	construct Constructor

	messages  map[string]*gridMessage
	nonceMsgs map[string]*gridMessage
}

var (
	_ Container               = (*GridContainer)(nil)
	_ cchat.MessagesContainer = (*GridContainer)(nil)
)

func NewGridContainer(constr Constructor) *GridContainer {
	grid, _ := gtk.GridNew()
	grid.SetColumnSpacing(10)
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
		messages:       map[string]*gridMessage{},
		nonceMsgs:      map[string]*gridMessage{},
	}

	return &container
}

func (c *GridContainer) Reset() {
	// does this actually work?
	var rows = c.len()
	for i := 0; i < rows; i++ {
		c.Main.RemoveRow(i)
	}

	c.messages = map[string]*gridMessage{}
	c.nonceMsgs = map[string]*gridMessage{}

	c.ScrolledWindow.Bottomed = true
}

func (c *GridContainer) len() int {
	return len(c.messages) + len(c.nonceMsgs)
}

// PresendMessage is not thread-safe.
func (c *GridContainer) PresendMessage(msg input.PresendMessage) func(error) {
	presend := c.construct.NewPresendMessage(msg)

	msgc := gridMessage{
		GridMessage: presend,
		presend:     presend,
		index:       c.len(),
	}

	c.nonceMsgs[presend.Nonce()] = &msgc
	msgc.Attach(c.Main, msgc.index)

	return func(err error) {
		if err != nil {
			presend.SetSentError(err)
			log.Error(errors.Wrap(err, "Failed to send message"))
		}
	}
}

// FindMessage is not thread-safe. This exists for backwards compatibility.
func (c *GridContainer) FindMessage(msg cchat.MessageHeader) GridMessage {
	if m := c.findMessage(msg); m != nil {
		return m.GridMessage
	}
	return nil
}

func (c *GridContainer) findMessage(msg cchat.MessageHeader) *gridMessage {
	// Search using the ID first.
	m, ok := c.messages[msg.ID()]
	if ok {
		return m
	}

	// Is this an existing message?
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		var nonce = noncer.Nonce()

		// Things in this map are guaranteed to have presend != nil.
		m, ok := c.nonceMsgs[nonce]
		if ok {
			// Move the message outside nonceMsgs.
			delete(c.nonceMsgs, nonce)
			c.messages[msg.ID()] = m

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
		// Attempt update before insert (aka upsert).
		if msgc := c.FindMessage(msg); msgc != nil {
			msgc.UpdateAuthor(msg.Author())
			msgc.UpdateContent(msg.Content())
			msgc.UpdateTimestamp(msg.Time())
			return
		}

		msgc := gridMessage{
			GridMessage: c.construct.NewMessage(msg),
			index:       c.len(),
		}

		c.messages[msgc.ID()] = &msgc
		msgc.Attach(c.Main, msgc.index)
	})
}

func (c *GridContainer) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() {
		if msgc := c.FindMessage(msg); msgc != nil {
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
		if m, ok := c.messages[msg.ID()]; ok {
			delete(c.messages, msg.ID())
			c.Main.RemoveRow(m.index)
		}
	})
}
