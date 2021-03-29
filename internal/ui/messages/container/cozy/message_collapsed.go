package cozy

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/gotk3/gotk3/gtk"
)

// Collapsed is a message that follows after FullMessage. It does not show
// the header, and the avatar is invisible.
type CollapsedMessage struct {
	// Author is still updated normally.
	*message.State
	Timestamp *gtk.Label
}

// WrapCollapsedMessage wraps the given message state to be a collapsed message.
func WrapCollapsedMessage(gc *message.State) *CollapsedMessage {
	// Set Timestamp's padding accordingly to Avatar's.
	ts := message.NewTimestamp()
	ts.SetSizeRequest(AvatarSize, -1)
	ts.SetVAlign(gtk.ALIGN_START)
	ts.SetXAlign(0.5) // middle align
	ts.SetMarginEnd(container.ColumnSpacing)
	ts.SetMarginStart(container.ColumnSpacing * 2)

	// Set Content's padding accordingly to FullMessage's main box.
	gc.Content.ToWidget().SetMarginEnd(container.ColumnSpacing * 2)

	gc.PackStart(ts, false, false, 0)
	gc.PackStart(gc.Content, true, true, 0)
	gc.SetClass("cozy-collapsed")

	return &CollapsedMessage{
		State:     gc,
		Timestamp: ts,
	}
}

func (c *CollapsedMessage) Collapsed() bool { return true }

func (c *CollapsedMessage) Unwrap(revert bool) *message.State {
	if revert {
		// Remove State's widgets from the containers.
		c.Remove(c.Timestamp)
		c.Remove(c.Content)
	}

	return c.State
}

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
