package cozy

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/gotk3/gotk3/gtk"
)

// Collapsed is a message that follows after FullMessage. It does not show
// the header, and the avatar is invisible.
type CollapsedMessage struct {
	// Author is still updated normally.
	*message.GenericContainer
}

func NewCollapsedMessage(msg cchat.MessageCreate) *CollapsedMessage {
	msgc := WrapCollapsedMessage(message.NewContainer(msg))
	message.FillContainer(msgc, msg)
	return msgc
}

func WrapCollapsedMessage(gc *message.GenericContainer) *CollapsedMessage {
	// Set Timestamp's padding accordingly to Avatar's.
	gc.Timestamp.SetSizeRequest(AvatarSize, -1)
	gc.Timestamp.SetVAlign(gtk.ALIGN_START)
	gc.Timestamp.SetXAlign(0.5) // middle align
	gc.Timestamp.SetMarginStart(container.ColumnSpacing * 2)
	gc.Timestamp.SetMarginTop(container.ColumnSpacing)

	// Set Content's padding accordingly to FullMessage's main box.
	gc.Content.ToWidget().SetMarginEnd(container.ColumnSpacing * 2)

	gc.PackStart(gc.Timestamp, false, false, 0)
	gc.PackStart(gc.Content, true, true, 0)
	gc.SetClass("cozy-collapsed")

	return &CollapsedMessage{
		GenericContainer: gc,
	}
}

func (c *CollapsedMessage) Collapsed() bool { return true }

func (c *CollapsedMessage) UpdateTimestamp(t time.Time) {
	c.GenericContainer.UpdateTimestamp(t)
	c.Timestamp.SetText(humanize.TimeAgoShort(t))
}

func (c *CollapsedMessage) Unwrap() *message.GenericContainer {
	// Remove GenericContainer's widgets from the containers.
	c.Remove(c.Timestamp)
	c.Remove(c.Content)

	// Return after removing.
	return c.GenericContainer
}

type CollapsedSendingMessage struct {
	*CollapsedMessage
	message.PresendContainer
}

func NewCollapsedSendingMessage(msg input.PresendMessage) *CollapsedSendingMessage {
	var msgc = message.NewPresendContainer(msg)

	return &CollapsedSendingMessage{
		CollapsedMessage: WrapCollapsedMessage(msgc.GenericContainer),
		PresendContainer: msgc,
	}
}
