package cozy

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
)

// Collapsed is a message that follows after FullMessage. It does not show
// the header, and the avatar is invisible.
type CollapsedMessage struct {
	// Author is still updated normally.
	*message.State
}

// WrapCollapsedMessage wraps the given message state to be a collapsed message.
func WrapCollapsedMessage(gc *message.State) *CollapsedMessage {
	// Set Content's padding accordingly to FullMessage's main box.
	gc.Content.SetMarginStart(container.ColumnSpacing*2 + AvatarSize)
	gc.Content.SetMarginEnd(container.ColumnSpacing)

	gc.PackStart(gc.Content, true, true, 0)
	gc.SetClass("cozy-collapsed")

	return &CollapsedMessage{
		State: gc,
	}
}

func (c *CollapsedMessage) Revert() *message.State {
	c.ClearBox()
	c.Content.SetMarginStart(0)
	c.Content.SetMarginEnd(0)
	return c.Unwrap()
}

type collapsed interface {
	collapsed()
}

func (c *CollapsedMessage) collapsed() {}

type CollapsedSendingMessage struct {
	*CollapsedMessage
	message.Presender
}

func WrapCollapsedSendingMessage(pstate *message.PresendState) *CollapsedSendingMessage {
	return &CollapsedSendingMessage{
		CollapsedMessage: WrapCollapsedMessage(pstate.State),
		Presender:        pstate,
	}
}
