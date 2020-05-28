package message

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*gtk.ScrolledWindow
	main     *gtk.Grid
	messages map[string]Message
}

func NewContainer() *Container {
	grid, _ := gtk.GridNew()
	grid.SetColumnSpacing(8)
	grid.SetRowSpacing(5)
	grid.SetMarginStart(5)
	grid.SetMarginEnd(5)
	grid.Show()

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Add(grid)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_ALWAYS)
	sw.Show()

	return &Container{sw, grid, map[string]Message{}}
}

func (c *Container) Reset() {
	// does this actually work?
	var rows = len(c.messages)
	for i := 0; i < rows; i++ {
		c.main.RemoveRow(i)
	}

	c.messages = nil
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		msgc := NewMessage(msg)
		msgc.index = len(c.messages) // unsure

		c.messages[msgc.ID] = msgc
		msgc.Attach(c.main, msgc.index)
	})
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() {
		mc, ok := c.messages[msg.ID()]
		if !ok {
			return
		}

		if author := msg.Author(); !author.Empty() {
			mc.UpdateAuthor(author)
		}
		if content := msg.Content(); !content.Empty() {
			mc.UpdateContent(content)
		}
	})
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() {
		if m, ok := c.messages[msg.ID()]; ok {
			delete(c.messages, msg.ID())
			c.main.RemoveRow(m.index)
		}
	})
}
