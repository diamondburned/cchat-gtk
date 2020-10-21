package container

import (
	"github.com/diamondburned/cchat"
	"github.com/google/btree"
)

// messageStore implements various data structures for optimized message get and
// insert.
type messageStore struct {
	msgTree       btree.BTree
	messageIDs    map[string]*gridMessage
	messageNonces map[string]*gridMessage
}

func newMessageStore() *messageStore {
	return &messageStore{
		msgTree:       *btree.New(2),
		messageIDs:    make(map[string]*gridMessage, 100),
		messageNonces: make(map[string]*gridMessage, 5),
	}
}

func (ms *messageStore) Len() int {
	return ms.msgTree.Len()
}

// InsertMessage inserts the message into the store and return the new index.
func (ms *messageStore) InsertMessage(msg *gridMessage) int {
	return ms.replaceMessage(msg, false)
}

// SwapMessage overrides the old message with the same ID with the given one. It
// returns an index if the message is replaced, or -1 if the message is not.
func (ms *messageStore) SwapMessage(msg *gridMessage) int {
	return ms.replaceMessage(msg, true)
}

func (ms *messageStore) replaceMessage(msg *gridMessage, replaceOnly bool) int {
	var ix = -1

	// Guarantee that no new messages are added.
	if replaced := ms.msgTree.ReplaceOrInsert(msg); replaced == nil && replaceOnly {
		// Nil is returned, meaning a new message is added. This is bad.
		ms.msgTree.Delete(msg)
		return ix
	}

	var id = msg.ID()
	if id != "" {
		ms.messageIDs[id] = msg
		delete(ms.messageNonces, msg.Nonce()) // superfluous guarantee
	} else {
		// Assume nonce is non-empty. Probably not a good idea.
		ms.messageNonces[msg.Nonce()] = msg
	}

	insertAt := ms.msgTree.Len() - 1

	ms.msgTree.Descend(func(item btree.Item) bool {
		if id == item.(*gridMessage).ID() {
			ix = insertAt
			return false // break
		}
		insertAt--
		return true
	})

	return ix
}

func (ms *messageStore) MessageBefore(id cchat.ID) GridMessage {
	return ms.getOffsetted(id, true)
}

func (ms *messageStore) MessageAfter(id cchat.ID) GridMessage {
	return ms.getOffsetted(id, false)
}

// getOffsetted returns the unwrapped message.
func (ms *messageStore) getOffsetted(id cchat.ID, before bool) GridMessage {
	var last, found *gridMessage
	var next bool

	// We need to ascend, as next and before implies ascending order from 0 to
	// last.
	ms.msgTree.Ascend(func(item btree.Item) bool {
		message := item.(*gridMessage)
		if next {
			found = message
			return false // break
		}
		if message.ID() == id {
			if before {
				found = last
				return false // break
			} else {
				next = true
				return true
			}
		}
		last = message
		return true
	})

	if found == nil {
		return nil
	}

	return found.GridMessage
}

// FirstMessage returns the earliest message.
func (ms *messageStore) FirstMessage() *gridMessage {
	return ms.msgTree.Min().(*gridMessage)
}

// LastMessage returns the latest message.
func (ms *messageStore) LastMessage() *gridMessage {
	return ms.msgTree.Max().(*gridMessage)
}

// NthMessage returns the nth message ordered from earliest to latest. It is
// fairly slow.
func (ms *messageStore) NthMessage(n int) (message *gridMessage) {
	insertAt := ms.msgTree.Len() - 1

	ms.msgTree.Descend(func(item btree.Item) bool {
		if n == insertAt {
			message = item.(*gridMessage)
			return false // break
		}
		insertAt--
		return true
	})

	return
}

// LastMessageFrom returns the latest message with the given user ID. This is
// used for the input prompt.
func (ms *messageStore) LastMessageFrom(userID string) GridMessage {
	return ms.FindMessage(func(gridMsg GridMessage) bool {
		return gridMsg.AuthorID() == userID
	})
}

// FindMessage implicitly unwraps the GridMessage before passing it into the
// handler and returning it.
func (ms *messageStore) FindMessage(isMessage func(GridMessage) bool) (found GridMessage) {
	ms.msgTree.Descend(func(item btree.Item) bool {
		message := item.(*gridMessage)
		// Ignore sending messages.
		if message.presend != nil {
			return true
		}
		if unwrapped := message.GridMessage; isMessage(unwrapped) {
			found = unwrapped
			return false
		}
		return true
	})
	return
}

func (ms *messageStore) get(id, nonce string) *gridMessage {
	if id != "" {
		m, ok := ms.messageIDs[id]
		if ok {
			return m
		}
	}

	m, ok := ms.messageNonces[nonce]
	if ok {
		return m
	}

	return nil
}

func (ms *messageStore) Message(msgID cchat.ID, nonce string) *gridMessage {
	message := ms.get(msgID, nonce)

	// If the message was obtained from a nonce, then try to move it off.
	if nonce != "" && message != nil {
		// Move the message from nonce state to ID.
		delete(ms.messageNonces, nonce)
		ms.messageIDs[msgID] = message

		// Set the right ID.
		message.presend.SetDone(msgID)
		// Destroy the presend struct.
		message.presend = nil
	}

	return message
}

func (ms *messageStore) DeleteMessage(msgID cchat.ID) {
	m, ok := ms.messageIDs[msgID]
	if ok {
		ms.msgTree.Delete(m)
		delete(ms.messageIDs, msgID)
	}
}

func (ms *messageStore) PopMessage(id cchat.ID) (popped *gridMessage, ix int) {
	ix = ms.msgTree.Len() - 1

	ms.msgTree.Descend(func(item btree.Item) bool {
		if gridMsg := item.(*gridMessage); id == gridMsg.ID() {
			popped = gridMsg

			// Delete off of the state.
			ms.msgTree.Delete(item)
			delete(ms.messageIDs, id)
			delete(ms.messageNonces, popped.Nonce()) // superfluous

			return false // break
		}
		ix--
		return true
	})

	return
}

// PopEarliestMessages pops the n earliest messages. n can be less than or equal
// to 0, which would be a no-op.
func (ms *messageStore) PopEarliestMessages(n int) (poppedIxs int) {
	for ; n > 0 && ms.Len() > 0; n-- {
		gridMsg := ms.msgTree.DeleteMin().(*gridMessage)
		delete(ms.messageIDs, gridMsg.ID())
		delete(ms.messageNonces, gridMsg.Nonce())

		// We can keep incrementing the index as we delete things. This is
		// because we're deleting from 0 and up.
		poppedIxs++
	}
	return
}
