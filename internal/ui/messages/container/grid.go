package container

import (
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

type GridStore struct {
	Grid *gtk.Grid

	Construct  Constructor
	Controller Controller

	messages   map[string]*gridMessage
	messageIDs []string // ids or nonces
}

func NewGridStore(constr Constructor, ctrl Controller) *GridStore {
	grid, _ := gtk.GridNew()
	grid.SetColumnSpacing(ColumnSpacing)
	grid.SetRowSpacing(5)
	grid.SetMarginStart(5)
	grid.SetMarginEnd(5)
	grid.SetMarginBottom(5)
	grid.Show()

	primitives.AddClass(grid, "message-grid")

	return &GridStore{
		Grid:       grid,
		Construct:  constr,
		Controller: ctrl,
		messages:   map[string]*gridMessage{},
	}
}

func (c *GridStore) Reset() {
	c.Grid.GetChildren().Foreach(func(v interface{}) {
		// Unsafe assertion ftw.
		w := v.(gtk.IWidget).ToWidget()
		c.Grid.Remove(w)
		w.Destroy()
	})

	c.messages = map[string]*gridMessage{}
	c.messageIDs = []string{}
}

func (c *GridStore) MessagesLen() int {
	return len(c.messages)
}

// findIndex searches backwards for idnonce.
func (c *GridStore) findIndex(idnonce string) int {
	for i := len(c.messageIDs) - 1; i >= 0; i-- {
		if c.messageIDs[i] == idnonce {
			return i
		}
	}
	return -1
}

// Swap changes the message with the ID to the given message. This provides a
// low level API for edits that need a new Attach method.
//
// TODO: combine compact and full so they share the same attach method.
func (c *GridStore) SwapMessage(msg GridMessage) bool {
	// Get the current message's index.
	var ix = c.findIndex(msg.ID())
	if ix == -1 {
		return false
	}

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.Grid.InsertRow(ix)

	// Let the new message be attached on top of the to-be-replaced message.
	msg.Attach(c.Grid, ix)

	// Delete the to-be-replaced message, which we have shifted downwards
	// earlier, so we add 1.
	c.Grid.RemoveRow(ix + 1)

	return true
}

// Before returns the message before the given ID, or nil if none.
func (c *GridStore) Before(id string) GridMessage {
	return c.getOffsetted(id, -1)
}

// After returns the message after the given ID, or nil if none.
func (c *GridStore) After(id string) GridMessage {
	return c.getOffsetted(id, 1)
}

func (c *GridStore) getOffsetted(id string, offset int) GridMessage {
	// Get the current index.
	var ix = c.findIndex(id)
	if ix == -1 {
		return nil
	}
	ix += offset

	if ix < 0 || ix >= len(c.messages) {
		return nil
	}

	return c.messages[c.messageIDs[ix]].GridMessage
}

// LatestMessageFrom returns the latest message with the given user ID. This is
// used for the input prompt.
func (c *GridStore) LatestMessageFrom(userID string) (msgID string, ok bool) {
	// FindMessage already looks from the latest messages.
	var msg = c.FindMessage(func(msg GridMessage) bool {
		return msg.AuthorID() == userID
	})

	if msg == nil {
		return "", false
	}

	return msg.ID(), true
}

// FindMessage iterates backwards and returns the message if isMessage() returns
// true on that message.
func (c *GridStore) FindMessage(isMessage func(msg GridMessage) bool) GridMessage {
	for i := len(c.messageIDs) - 1; i >= 0; i-- {
		msg := c.messages[c.messageIDs[i]]
		// Ignore sending messages.
		if msg.presend != nil {
			continue
		}
		// Check.
		if msg := msg.GridMessage; isMessage(msg) {
			return msg
		}
	}
	return nil
}

// FirstMessage returns the first message.
func (c *GridStore) FirstMessage() GridMessage {
	if len(c.messageIDs) > 0 {
		return c.messages[c.messageIDs[0]].GridMessage
	}
	return nil
}

// LastMessage returns the latest message.
func (c *GridStore) LastMessage() GridMessage {
	if l := len(c.messageIDs); l > 0 {
		return c.messages[c.messageIDs[l-1]].GridMessage
	}
	return nil
}

// Message finds the message state in the container. It is not thread-safe. This
// exists for backwards compatibility.
func (c *GridStore) Message(msg cchat.MessageHeader) GridMessage {
	if m := c.message(msg); m != nil {
		return m.GridMessage
	}
	return nil
}

func (c *GridStore) message(msg cchat.MessageHeader) *gridMessage {
	// Search using the ID first.
	m, ok := c.messages[msg.ID()]
	if ok {
		return m
	}

	// Is this an existing message?
	if noncer, ok := msg.(cchat.MessageNonce); ok {
		var nonce = noncer.Nonce()

		// Things in this map are guaranteed to have presend != nil.
		m, ok := c.messages[nonce]
		if ok {
			// Replace the nonce key with ID.
			delete(c.messages, nonce)
			c.messages[msg.ID()] = m

			// Set the right ID.
			m.presend.SetDone(msg.ID())
			// Destroy the presend struct.
			m.presend = nil

			// Replace the nonce inside the ID slice with the actual ID.
			if ix := c.findIndex(nonce); ix > -1 {
				c.messageIDs[ix] = msg.ID()
			} else {
				log.Error(fmt.Errorf("Missed ID %s in slice index %d", msg.ID(), ix))
			}

			return m
		}
	}

	return nil
}

// AddPresendMessage inserts an input.PresendMessage into the container and
// returning a wrapped widget interface.
func (c *GridStore) AddPresendMessage(msg input.PresendMessage) PresendGridMessage {
	presend := c.Construct.NewPresendMessage(msg)

	msgc := &gridMessage{
		GridMessage: presend,
		presend:     presend,
	}

	// Set the message into the grid.
	msgc.Attach(c.Grid, c.MessagesLen())
	// Append the NONCE.
	c.messageIDs = append(c.messageIDs, msgc.Nonce())
	// Set the NONCE into the message map.
	c.messages[msgc.Nonce()] = msgc

	return presend
}

func (c *GridStore) CreateMessageUnsafe(msg cchat.MessageCreate) {
	// Attempt to update before insertion (aka upsert).
	if msgc := c.Message(msg); msgc != nil {
		msgc.UpdateAuthor(msg.Author())
		msgc.UpdateContent(msg.Content(), false)
		msgc.UpdateTimestamp(msg.Time())

		c.Controller.BindMenu(msgc)
		return
	}

	msgc := &gridMessage{
		GridMessage: c.Construct.NewMessage(msg),
	}

	// Copy from PresendMessage.
	msgc.Attach(c.Grid, c.MessagesLen())
	c.messageIDs = append(c.messageIDs, msgc.ID())
	c.messages[msgc.ID()] = msgc

	c.Controller.BindMenu(msgc)
}

func (c *GridStore) UpdateMessageUnsafe(msg cchat.MessageUpdate) {
	if msgc := c.Message(msg); msgc != nil {
		if author := msg.Author(); author != nil {
			msgc.UpdateAuthor(author)
		}
		if content := msg.Content(); !content.Empty() {
			msgc.UpdateContent(content, true)
		}
	}

	return
}

func (c *GridStore) DeleteMessageUnsafe(msg cchat.MessageDelete) {
	c.PopMessage(msg.ID())
}

// PopMessage deletes a message off of the list and return the deleted message.
func (c *GridStore) PopMessage(id string) (msg GridMessage) {
	// Search for the index.
	var ix = c.findIndex(id)
	if ix < 0 {
		return nil
	}

	// Grab the message before deleting.
	msg = c.messages[id]

	// Remove off of the Gtk grid.
	c.Grid.RemoveRow(ix)
	// Pop off the slice.
	c.messageIDs = append(c.messageIDs[:ix], c.messageIDs[ix+1:]...)
	// Delete off the map.
	delete(c.messages, id)

	return
}
