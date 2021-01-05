package cozy

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
)

// Unwrapper provides an interface for messages to be unwrapped. This is used to
// convert between collapsed and full messages.
type Unwrapper interface {
	Unwrap() *message.GenericContainer
}

var (
	_ Unwrapper = (*CollapsedMessage)(nil)
	_ Unwrapper = (*CollapsedSendingMessage)(nil)
	_ Unwrapper = (*FullMessage)(nil)
	_ Unwrapper = (*FullSendingMessage)(nil)
)

// Collapsible is an interface for cozy messages to return whether or not
// they're full or collapsed.
type Collapsible interface {
	// Compact returns true if the message is a compact one and not full.
	Collapsed() bool
}

var (
	_ Collapsible = (*CollapsedMessage)(nil)
	_ Collapsible = (*CollapsedSendingMessage)(nil)
	_ Collapsible = (*FullMessage)(nil)
	_ Collapsible = (*FullSendingMessage)(nil)
)

const (
	AvatarSize   = 40
	AvatarMargin = 10
)

var messageConstructors = container.Constructor{
	NewMessage:        NewMessage,
	NewPresendMessage: NewPresendMessage,
}

func NewMessage(
	msg cchat.MessageCreate, before container.MessageRow) container.MessageRow {

	if gridMessageIsAuthor(before, msg.Author()) {
		return NewCollapsedMessage(msg)
	}

	return NewFullMessage(msg)
}

func NewPresendMessage(
	msg input.PresendMessage, before container.MessageRow) container.PresendMessageRow {

	if gridMessageIsAuthor(before, msg.Author()) {
		return NewCollapsedSendingMessage(msg)
	}

	return NewFullSendingMessage(msg)
}

type Container struct {
	*container.ListContainer
}

func NewContainer(ctrl container.Controller) *Container {
	c := container.NewListContainer(ctrl, messageConstructors)
	primitives.AddClass(c, "cozy-container")
	return &Container{ListContainer: c}
}

func (c *Container) findAuthorID(authorID string) container.MessageRow {
	// Search the old author if we have any.
	return c.ListStore.FindMessage(func(msgc container.MessageRow) bool {
		return msgc.Author().ID() == authorID
	})
}

// reuseAvatar tries to search past messages with the same author ID and URL for
// the image. It will fetch anew if there's none.
func (c *Container) reuseAvatar(authorID, avatarURL string, full *FullMessage) {
	// Is this a message that we can work with? We have to assert to
	// FullSendingMessage because that's where our messages are.
	var lastAuthorMsg = c.findAuthorID(authorID)

	// Borrow the avatar pixbuf, but only if the avatar URL is the same.
	p, ok := lastAuthorMsg.(AvatarPixbufCopier)
	if ok && lastAuthorMsg.Author().Avatar() == avatarURL {
		if p.CopyAvatarPixbuf(full.Avatar.Image) {
			full.Avatar.ManuallySetURL(avatarURL)
			return
		}
	}

	// We can't borrow, so we need to fetch it anew.
	full.Avatar.SetURL(avatarURL)
}

// lastMessageIsAuthor removed - assuming index before insertion is harmful.

func gridMessageIsAuthor(gridMsg container.MessageRow, author cchat.Author) bool {
	if gridMsg == nil {
		return false
	}
	leftAuthor := gridMsg.Author()
	return true &&
		leftAuthor.ID() == author.ID() &&
		leftAuthor.Name().String() == author.Name().String()
}

func (c *Container) CreateMessage(msg cchat.MessageCreate) {
	gts.ExecAsync(func() {
		// Create the message in the parent's handler. This handler will also
		// wipe old messages.
		row := c.ListContainer.CreateMessageUnsafe(msg)

		// Is this a full message? If so, then we should fetch the avatar when
		// we can.
		if full, ok := row.(*FullMessage); ok {
			author := msg.Author()
			avatarURL := author.Avatar()

			// Try and reuse an existing avatar if the author has one.
			if avatarURL != "" {
				// Try reusing the avatar, but fetch it from the internet if we can't
				// reuse. The reuse function does this for us.
				c.reuseAvatar(author.ID(), avatarURL, full)
			}
		}

		// Did the handler wipe old messages? It will only do so if the user is
		// scrolled to the bottom.
		if c.ListContainer.CleanMessages() {
			// We need to uncollapse the first (top) message. No length check is
			// needed here, as we just inserted a message.
			c.uncompact(c.FirstMessage())
		}

		// If we've prepended the message, then see if we need to collapse the
		// second message.
		if first := c.ListContainer.FirstMessage(); first != nil && first.ID() == msg.ID() {
			// If the author is the same, then collapse.
			if sec := c.NthMessage(1); sec != nil && gridMessageIsAuthor(sec, msg.Author()) {
				c.compact(sec)
			}
		}
	})
}

func (c *Container) UpdateMessage(msg cchat.MessageUpdate) {
	gts.ExecAsync(func() {
		c.UpdateMessageUnsafe(msg)
	})
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

		msgAuthorID := msg.Author().ID()

		// Check if the last message is the author's (relative to i):
		if prev.Author().ID() == msgAuthorID {
			// If the author is the same, then we don't need to uncollapse the
			// message.
			return
		}

		// If the next message (relative to i) is not the deleted message's
		// author, then we don't need to uncollapse it.
		if next.Author().ID() != msgAuthorID {
			return
		}

		// Uncompact or turn the message to a full one.
		c.uncompact(next)
	})
}

func (c *Container) uncompact(msg container.MessageRow) {
	// We should only uncompact the message if it's compacted in the first
	// place.
	compact, ok := msg.(*CollapsedMessage)
	if !ok {
		return
	}

	// Start the "lengthy" uncollapse process.
	full := WrapFullMessage(compact.Unwrap())
	// Update the container to reformat everything including the timestamps.
	message.RefreshContainer(full, full.GenericContainer)
	// Update the avatar if needed be, since we're now showing it.
	author := msg.Author()
	c.reuseAvatar(author.ID(), author.Avatar(), full)

	// Swap the old next message out for a new one.
	c.ListStore.SwapMessage(full)
}

func (c *Container) compact(msg container.MessageRow) {
	full, ok := msg.(*FullMessage)
	if !ok {
		return
	}

	compact := WrapCollapsedMessage(full.Unwrap())
	message.RefreshContainer(compact, compact.GenericContainer)

	c.ListStore.SwapMessage(compact)
}
