package container

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/google/btree"
)

// gridMessage w/ required internals
type gridMessage struct {
	GridMessage
	presend message.PresendContainer // this shouldn't be here but i'm lazy
}

// Less compares the time while accounting for equal time with different IDs.
func (g *gridMessage) Less(than btree.Item) bool {
	thanMessage := than.(*gridMessage)

	// Time must never match if the IDs don't.
	if thanMessage.Time().Equal(g.Time()) {
		if thanMessage.ID() != g.ID() {
			// Always return less = true because this shouldn't be equal.
			return true
		}
	}

	return g.Time().Before(thanMessage.Time())
}

// unwrap returns nil if g is nil. Otherwise, it unwraps the gridMessage.
func (g *gridMessage) unwrap() GridMessage {
	if g == nil {
		return nil
	}
	return g.GridMessage
}
