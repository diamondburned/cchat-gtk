package cozy

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

// Unwrapper provides an interface for messages to be unwrapped. This is used to
// convert between collapsed and full messages.
type Unwrapper interface {
	Unwrap(grid *gtk.Grid) *message.GenericContainer
}

var (
	_ Unwrapper = (*CollapsedMessage)(nil)
	_ Unwrapper = (*CollapsedSendingMessage)(nil)
	_ Unwrapper = (*FullMessage)(nil)
	_ Unwrapper = (*FullSendingMessage)(nil)
)

const (
	AvatarSize   = 40
	AvatarMargin = 10
)

type Container struct {
	*container.GridContainer
}

func NewContainer(ctrl container.Controller) *Container {
	c := &Container{}
	c.GridContainer = container.NewGridContainer(c, ctrl)
	// A not-so-generous row padding, as we will rely on margins per widget.
	c.GridContainer.Grid.SetRowSpacing(2)

	primitives.AddClass(c, "cozy-container")
	return c
}

func (c *Container) NewMessage(msg cchat.MessageCreate) container.GridMessage {
	// Is the latest message of the same author? If yes, display it as a
	// collapsed message.
	if c.lastMessageIsAuthor(msg.Author().ID()) {
		return NewCollapsedMessage(msg)
	}

	full := NewFullMessage(msg)
	author := msg.Author()

	// Try and reuse an existing avatar if the author has one.
	if avatarURL, ok := author.(cchat.MessageAuthorAvatar); ok {
		// Try reusing the avatar, but fetch it from the interndet if we can't
		// reuse. The reuse function does this for us.
		c.reuseAvatar(author.ID(), avatarURL.Avatar(), full)
	}

	return full
}

func (c *Container) NewPresendMessage(msg input.PresendMessage) container.PresendGridMessage {
	if c.lastMessageIsAuthor(msg.AuthorID()) {
		return NewCollapsedSendingMessage(msg)
	}

	full := NewFullSendingMessage(msg)

	// Try and see if we can reuse the avatar, and fallback if possible. The
	// avatar URL passed in here will always yield an equal.
	c.reuseAvatar(msg.AuthorID(), msg.AuthorAvatarURL(), &full.FullMessage)

	return full
}

func (c *Container) lastMessageIsAuthor(id string) bool {
	var last = c.GridStore.LastMessage()
	return last != nil && last.AuthorID() == id
}

func (c *Container) findAuthorID(authorID string) container.GridMessage {
	// Search the old author if we have any.
	return c.GridStore.FindMessage(func(msgc container.GridMessage) bool {
		return msgc.AuthorID() == authorID
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
	if ok && lastAuthorMsg.AvatarURL() == avatarURL {
		p.CopyAvatarPixbuf(full.Avatar)
		full.Avatar.ManuallySetURL(avatarURL)
	} else {
		// We can't borrow, so we need to fetch it anew.
		full.Avatar.SetURL(avatarURL)
	}
}

func (c *Container) DeleteMessage(msg cchat.MessageDelete) {
	gts.ExecAsync(func() {
		// Get the previous and next message before deleting. We'll need them to
		// evaluate whether we need to change anything.
		prev := c.GridStore.Before(msg.ID())
		next := c.GridStore.After(msg.ID())

		// The function doesn't actually try and re-collapse the bottom message
		// when a sandwiched message is deleted. This is fine.

		// Delete the message off of the parent's container.
		msg := c.GridStore.PopMessage(msg.ID())

		// Don't calculate if we don't have any messages, or no messages before
		// and after.
		if c.GridStore.MessagesLen() == 0 || prev == nil || next == nil {
			return
		}

		// Check if the last message is the author's (relative to i):
		if prev.AuthorID() == msg.AuthorID() {
			// If the author is the same, then we don't need to uncollapse the
			// message.
			return
		}

		// If the next message (relative to i) is not the deleted message's
		// author, then we don't need to uncollapse it.
		if next.AuthorID() != msg.AuthorID() {
			return
		}

		// Get the unwrapper method, which allows us to get the
		// *message.GenericContainer.
		uw, ok := next.(Unwrapper)
		if !ok {
			return
		}

		// Start the "lengthy" uncollapse process.
		full := WrapFullMessage(uw.Unwrap(c.Grid))
		// Update the container to reformat everything including the timestamps.
		message.RefreshContainer(full, full.GenericContainer)
		// Update the avatar if needed be, since we're now showing it.
		c.reuseAvatar(next.AuthorID(), next.AvatarURL(), full)

		// Swap the old next message out for a new one.
		c.GridStore.SwapMessage(full)
	})
}
