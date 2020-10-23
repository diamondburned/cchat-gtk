package container

import (
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type GridStore struct {
	*gtk.Grid

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
	grid.Show()

	primitives.AddClass(grid, "message-grid")

	return &GridStore{
		Grid:       grid,
		Construct:  constr,
		Controller: ctrl,
		messages:   map[string]*gridMessage{},
	}
}

func (c *GridStore) MessagesLen() int {
	return len(c.messages)
}

func (c *GridStore) attachGrid(row int, widgets []gtk.IWidget) {
	for i, w := range widgets {
		c.Grid.Attach(w, i, row, 1, 1)
	}
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

type CoordinateTranslator interface {
	TranslateCoordinates(dest gtk.IWidget, srcX int, srcY int) (destX int, destY int, e error)
}

var _ CoordinateTranslator = (*gtk.Widget)(nil)

func (c *GridStore) TranslateCoordinates(parent gtk.IWidget, msg GridMessage) (y int) {
	i := c.findIndex(msg.ID())
	if i < 0 {
		return 0
	}

	m, _ := c.messages[c.messageIDs[i]]
	w, _ := m.Focusable().(CoordinateTranslator)

	// x is not needed.
	_, y, err := w.TranslateCoordinates(parent, 0, 0)
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to translate coords while focusing"))
		return
	}

	// log.Println("X:", x)
	// log.Println("Y:", y)

	return y
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

	// Wrap msg inside a *gridMessage if it's not already.
	mg, ok := msg.(*gridMessage)
	if !ok {
		mg = &gridMessage{GridMessage: msg}
	}

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.Grid.InsertRow(ix)

	// Let the new message be attached on top of the to-be-replaced message.
	c.attachGrid(ix, mg.Attach())

	// Set the message into the map.
	c.messages[mg.ID()] = mg

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

// NthMessage returns the nth message.
func (c *GridStore) NthMessage(n int) GridMessage {
	if len(c.messageIDs) > 0 && n >= 0 && n < len(c.messageIDs) {
		return c.messages[c.messageIDs[n]].GridMessage
	}
	return nil
}

// FirstMessage returns the first message.
func (c *GridStore) FirstMessage() GridMessage {
	return c.NthMessage(0)
}

// LastMessage returns the latest message.
func (c *GridStore) LastMessage() GridMessage {
	return c.NthMessage(c.MessagesLen() - 1)
}

// Message finds the message state in the container. It is not thread-safe. This
// exists for backwards compatibility.
func (c *GridStore) Message(msgID cchat.ID, nonce string) GridMessage {
	if m := c.message(msgID, nonce); m != nil {
		return m.GridMessage
	}
	return nil
}

func (c *GridStore) message(msgID cchat.ID, nonce string) *gridMessage {
	// Search using the ID first.
	m, ok := c.messages[msgID]
	if ok {
		return m
	}

	// Is this an existing message?
	if nonce != "" {
		// Things in this map are guaranteed to have presend != nil.
		m, ok := c.messages[nonce]
		if ok {
			// Replace the nonce key with ID.
			delete(c.messages, nonce)
			c.messages[msgID] = m

			// Set the right ID.
			m.presend.SetDone(msgID)
			// Destroy the presend struct.
			m.presend = nil

			// Replace the nonce inside the ID slice with the actual ID.
			if ix := c.findIndex(nonce); ix > -1 {
				c.messageIDs[ix] = msgID
			} else {
				log.Error(fmt.Errorf("Missed ID %s in slice index %d", msgID, ix))
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
	c.attachGrid(c.MessagesLen(), msgc.Attach())
	// Append the NONCE.
	c.messageIDs = append(c.messageIDs, msgc.Nonce())
	// Set the NONCE into the message map.
	c.messages[msgc.Nonce()] = msgc

	return presend
}

func (c *GridStore) PrependMessageUnsafe(msg cchat.MessageCreate) {
	msgc := &gridMessage{
		GridMessage: c.Construct.NewMessage(msg),
	}

	c.Grid.InsertRow(0)
	c.attachGrid(0, msgc.Attach())

	// Prepend the message ID.
	c.messageIDs = append(c.messageIDs, "")
	copy(c.messageIDs[1:], c.messageIDs)
	c.messageIDs[0] = msgc.ID()

	// Set the message into the map.
	c.messages[msgc.ID()] = msgc

	c.Controller.BindMenu(msgc)
}

func (c *GridStore) CreateMessageUnsafe(msg cchat.MessageCreate) {
	// Call the event handler last.
	defer c.Controller.AuthorEvent(msg.Author())

	// Attempt to update before insertion (aka upsert).
	if msgc := c.Message(msg.ID(), msg.Nonce()); msgc != nil {
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
	c.attachGrid(c.MessagesLen(), msgc.Attach())
	c.messageIDs = append(c.messageIDs, msgc.ID())
	c.messages[msgc.ID()] = msgc

	c.Controller.BindMenu(msgc)
}

func (c *GridStore) UpdateMessageUnsafe(msg cchat.MessageUpdate) {
	// Call the event handler last.
	defer c.Controller.AuthorEvent(msg.Author())

	if msgc := c.Message(msg.ID(), ""); msgc != nil {
		if author := msg.Author(); author != nil {
			msgc.UpdateAuthor(author)
		}
		if content := msg.Content(); !content.IsEmpty() {
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
