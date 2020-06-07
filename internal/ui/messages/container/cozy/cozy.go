package cozy

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
)

type AvatarPixbufCopier interface {
	CopyAvatarPixbuf(httputil.ImageContainer)
}

const (
	AvatarSize   = 40
	AvatarMargin = 10
)

type Container struct {
	*container.GridContainer
}

func NewContainer() *Container {
	c := &Container{}
	c.GridContainer = container.NewGridContainer(c)
	return c
}

func (c *Container) NewMessage(msg cchat.MessageCreate) container.GridMessage {
	var newmsg = NewFullMessage(msg)

	// Try and reuse an existing avatar.
	if author := msg.Author(); !c.reuseAvatar(author.ID(), newmsg.Avatar) {
		// Fetch a new avatar if we can't reuse the old one.
		newmsg.updateAuthorAvatar(author)
	}

	return newmsg
}

func (c *Container) NewPresendMessage(msg input.PresendMessage) container.PresendGridMessage {
	var presend = NewFullSendingMessage(msg)

	// Try and see if we can reuse the avatar, and fallback if possible.
	if !c.reuseAvatar(msg.AuthorID(), presend.Avatar) {
		presend.overrideAuthorAvatar(msg.AuthorAvatarURL())
	}

	return presend
}

func (c *Container) reuseAvatar(authorID string, img httputil.ImageContainer) (reused bool) {
	// Search the old author if we have any.
	msgc := c.FindMessage(func(msgc container.GridMessage) bool {
		return msgc.AuthorID() == authorID
	})

	// Is this a message that we can work with? We have to assert to
	// FullSendingMessage because that's where our messages are.
	copier, ok := msgc.(AvatarPixbufCopier)
	if ok {
		// Borrow the avatar URL.
		copier.CopyAvatarPixbuf(img)
	}

	return ok
}
