package cozy

// CompactMessage is a message that follows after FullMessage. It does not show
// the header, and the avatar is invisible.
type CompactMessage struct {
	// Essentially, CompactMessage is just a full message with some things
	// hidden. Its Avatar and Timestamp will still be updated. This is a
	// trade-off between performance, efficiency and code length.

	*FullMessage
}
