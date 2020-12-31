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

	// Set Content's padding accordingly to FullMessage's main box.
	gc.Content.ToWidget().SetMarginEnd(container.ColumnSpacing * 2)

	gc.Username.SetMaxWidthChars(30)

	return &CollapsedMessage{
		GenericContainer: gc,
	}
}

func (c *CollapsedMessage) Collapsed() bool { return true }

func (c *CollapsedMessage) UpdateTimestamp(t time.Time) {
	c.GenericContainer.UpdateTimestamp(t)
	c.Timestamp.SetText(humanize.TimeAgoShort(t))
}

func (c *CollapsedMessage) Unwrap(grid *gtk.Grid) *message.GenericContainer {
	// Remove GenericContainer's widgets from the containers.
	grid.Remove(c.Timestamp)
	grid.Remove(c.Content)

	// Return after removing.
	return c.GenericContainer
}

func (c *CollapsedMessage) Attach() []gtk.IWidget {
	return []gtk.IWidget{c.Timestamp, c.Content}
}

func (c *CollapsedMessage) Focusable() gtk.IWidget {
	return c.Timestamp
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
