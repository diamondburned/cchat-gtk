package message

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*gtk.ScrolledWindow
	main     *gtk.Box
	messages map[string]Message
}

func NewContainer() *Container {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 3)
	box.Show()

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Show()

	return &Container{sw, box, map[string]Message{}}
}

func (c *Container) Reset() {
	for _, msg := range c.messages {
		c.main.Remove(msg)
	}

	c.messages = nil
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		var msgc = NewMessage(msg)
		c.messages[msgc.ID] = msgc
		c.main.Add(msgc)
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
			c.main.Remove(m)
		}
	})
}
