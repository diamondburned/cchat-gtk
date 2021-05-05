package cozy

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
)

const AvatarSize = message.AvatarSize

// NewMessage creates a new message.
func NewMessage(
	s *message.State, before container.MessageRow) container.MessageRow {

	if isCollapsible(before, s) {
		return WrapCollapsedMessage(s)
	}

	return WrapFullMessage(s)
}

// NewPresendMessage creates a new presend message.
func NewPresendMessage(
	s *message.PresendState, before container.MessageRow) container.PresendMessageRow {

	if isCollapsible(before, s.State) {
		return WrapCollapsedSendingMessage(s)
	}

	return WrapFullSendingMessage(s)
}

type Container struct {
	*container.ListContainer
}

func NewContainer(ctrl container.Controller) *Container {
	c := container.NewListContainer(ctrl)
	primitives.AddClass(c, "cozy-container")
	return &Container{ListContainer: c}
}

const splitDuration = 3 * time.Minute

// isCollapsible returns true if the given lastMsg has matching conditions with
// the given msg.
func isCollapsible(last container.MessageRow, msg *message.State) bool {
	if last == nil || msg == nil {
		return false
	}

	lastMsg := last.Unwrap()

	return true &&
		lastMsg.Author.ID == msg.Author.ID &&
		lastMsg.Time.Add(splitDuration).After(msg.Time)
}

func (c *Container) NewPresendMessage(state *message.PresendState) container.PresendMessageRow {
	before, at := container.InsertPosition(c, state.Time)
	msgr := NewPresendMessage(state, before)
	c.AddMessageAt(msgr, at)
	return msgr
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		before, at := container.InsertPosition(c, msg.Time())
		state := message.NewState(msg)
		msgr := NewMessage(state, before)
		c.AddMessageAt(msgr, at)
	})
}

// AddMessage adds the given message.
func (c *Container) AddMessageAt(msgr container.MessageRow, ix int) {
	// Create the message in the parent's handler. This handler will also
	// wipe old messages.
	c.ListContainer.AddMessageAt(msgr, ix)

	// Did the handler wipe old messages? It will only do so if the user is
	// scrolled to the bottom.
	if c.ListContainer.CleanMessages() {
		// We need to uncollapse the first (top) message. No length check is
		// needed here, as we just inserted a message.
		c.uncompact(container.FirstMessage(c))
	}

	// If we've prepended the message, then see if we need to collapse the
	// second message.
	if ix == -1 {
		// If the author is the same, then collapse.
		if sec := c.NthMessage(1); isCollapsible(sec, msgr.Unwrap()) {
			c.compact(sec)
		}
	}
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() { container.UpdateMessage(c, msg) })
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() {
		msgID := msg.ID()

		// Get the previous and next message before deleting. We'll need them to
		// evaluate whether we need to change anything.
		prev, next := c.ListStore.Around(msgID)

		// The function doesn't actually try and re-collapse the bottom message
		// when a sandwiched message is deleted. This is fine.

		// Delete the message off of the parent's container.
		msg := c.ListStore.PopMessage(msgID)

		// Don't calculate if we don't have any messages, or no messages before
		// and after.
		if c.ListStore.MessagesLen() == 0 || prev == nil || next == nil {
			return
		}

		msgHeader := msg.Unwrap()

		prevHeader := prev.Unwrap()
		nextHeader := next.Unwrap()

		// Check if the last message is the author's (relative to i):
		if prevHeader.Author.ID == msgHeader.Author.ID {
			// If the author is the same, then we don't need to uncollapse the
			// message.
			return
		}

		// If the next message (relative to i) is not the deleted message's
		// author, then we don't need to uncollapse it.
		if nextHeader.Author.ID != msgHeader.Author.ID {
			return
		}

		// Uncompact or turn the message to a full one.
		c.uncompact(next)
	})
}

func (c *Container) uncompact(msg container.MessageRow) {
	_, isFull := msg.(full)
	if isFull {
		return
	}

	full := WrapFullMessage(msg.Revert())
	c.ListStore.SwapMessage(full)
}

func (c *Container) compact(msg container.MessageRow) {
	_, isCollapsed := msg.(collapsed)
	if isCollapsed {
		return
	}

	compact := WrapCollapsedMessage(msg.Revert())
	c.ListStore.SwapMessage(compact)
}
