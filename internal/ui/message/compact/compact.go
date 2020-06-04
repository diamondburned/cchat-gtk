package compact

import (
	"fmt"
	"html"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/message/autoscroll"
	"github.com/diamondburned/cchat-gtk/internal/ui/message/input"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*autoscroll.ScrolledWindow
	main      *gtk.Grid
	messages  map[string]*Message
	nonceMsgs map[string]*Message

	bottomed bool
}

func NewContainer() *Container {
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

	container := Container{
		ScrolledWindow: sw,
		main:           grid,
		messages:       map[string]*Message{},
		nonceMsgs:      map[string]*Message{},
		bottomed:       true, // bottomed by default.
	}

	return &container
}

func (c *Container) Reset() {
	// does this actually work?
	var rows = c.len()
	for i := 0; i < rows; i++ {
		c.main.RemoveRow(i)
	}

	c.messages = map[string]*Message{}
	c.nonceMsgs = map[string]*Message{}

	// default to being bottomed
	c.bottomed = true
}

func (c *Container) len() int {
	return len(c.messages) + len(c.nonceMsgs)
}

// PresendMessage is not thread-safe.
func (c *Container) PresendMessage(msg input.PresendMessage) func(error) {
	msgc := NewPresendMessage(msg.Content(), msg.Author(), msg.AuthorID(), msg.Nonce())
	msgc.index = c.len()

	c.nonceMsgs[msgc.Nonce] = &msgc
	msgc.Attach(c.main, msgc.index)

	return func(err error) {
		msgc.SetSensitive(true)

		// Did we fail?
		if err != nil {
			msgc.Content.SetMarkup(fmt.Sprintf(
				`<span color="red">%s</span>`,
				html.EscapeString(msgc.Content.GetLabel()),
			))
			msgc.Content.SetTooltipText(err.Error())
		}
	}
}

// FindMessage is not thread-safe.
func (c *Container) FindMessage(msg cchat.MessageHeader) *Message {
	// Search using the ID first.
	m, ok := c.messages[msg.ID()]
	if ok {
		return m
	}

	// Is this an existing message?
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		var nonce = noncer.Nonce()

		m, ok := c.nonceMsgs[nonce]
		if ok {
			// Move the message outside nonceMsgs.
			delete(c.nonceMsgs, nonce)
			c.messages[msg.ID()] = m

			// Set the right ID.
			m.ID = msg.ID()

			return m
		}
	}

	return nil
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		// Attempt update before insert (aka upsert).
		if msgc := c.FindMessage(msg); msgc != nil {
			msgc.SetSensitive(true)
			msgc.UpdateAuthor(msg.Author())
			msgc.UpdateContent(msg.Content())
			msgc.UpdateTimestamp(msg.Time())
			return
		}

		msgc := NewMessage(msg)
		msgc.index = c.len() // unsure

		c.messages[msgc.ID] = &msgc
		msgc.Attach(c.main, msgc.index)
	})
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
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

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() {
		// TODO: add nonce check.
		if m, ok := c.messages[msg.ID()]; ok {
			delete(c.messages, msg.ID())
			c.main.RemoveRow(m.index)
		}
	})
}
