package container

import (
	"container/list"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type messageKey struct {
	id    string
	nonce bool
}

func nonceKey(nonce string) messageKey { return messageKey{nonce, true} }
func idKey(id cchat.ID) messageKey     { return messageKey{id, false} }

type GridStore struct {
	*gtk.Grid

	Construct  Constructor
	Controller Controller

	resetMe bool

	messages    map[messageKey]*gridMessage
	messageList *list.List
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
		Grid:        grid,
		Construct:   constr,
		Controller:  ctrl,
		messages:    make(map[messageKey]*gridMessage, BacklogLimit+1),
		messageList: list.New(),
	}
}

func (c *GridStore) Reset() {
	primitives.RemoveChildren(c.Grid)
	c.messages = make(map[messageKey]*gridMessage, BacklogLimit+1)
	c.messageList = list.New()
}

func (c *GridStore) MessagesLen() int {
	return c.messageList.Len()
}

func (c *GridStore) attachGrid(row int, widgets []gtk.IWidget) {
	for i, w := range widgets {
		c.Grid.Attach(w, i, row, 1, 1)
	}
}

func (c *GridStore) findElement(id cchat.ID) (*list.Element, *gridMessage, int) {
	var index = c.messageList.Len() - 1
	for elem := c.messageList.Back(); elem != nil; elem = elem.Prev() {
		if gridMsg := elem.Value.(*gridMessage); gridMsg.ID() == id {
			return elem, gridMsg, index
		}
		index--
	}
	return nil, nil, -1
}

// findIndex searches backwards for id.
func (c *GridStore) findIndex(id cchat.ID) (*gridMessage, int) {
	_, gridMsg, ix := c.findElement(id)
	return gridMsg, ix
}

type CoordinateTranslator interface {
	TranslateCoordinates(dest gtk.IWidget, srcX int, srcY int) (destX int, destY int, e error)
}

var _ CoordinateTranslator = (*gtk.Widget)(nil)

func (c *GridStore) TranslateCoordinates(parent gtk.IWidget, msg GridMessage) (y int) {
	m, i := c.findIndex(msg.ID())
	if i < 0 {
		return 0
	}

	w, _ := m.Focusable().(CoordinateTranslator)

	// x is not needed.
	_, y, err := w.TranslateCoordinates(parent, 0, 0)
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to translate coords while focusing"))
		return
	}

	return y
}

// Swap changes the message with the ID to the given message. This provides a
// low level API for edits that need a new Attach method.
//
// TODO: combine compact and full so they share the same attach method.
func (c *GridStore) SwapMessage(msg GridMessage) bool {
	// Wrap msg inside a *gridMessage if it's not already.
	m, ok := msg.(*gridMessage)
	if !ok {
		m = &gridMessage{GridMessage: msg}
	}

	// Get the current message's index.
	_, ix := c.findIndex(msg.ID())
	if ix == -1 {
		return false
	}

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.Grid.InsertRow(ix)

	// Delete the to-be-replaced message, which we have shifted downwards
	// earlier, so we add 1.
	c.Grid.RemoveRow(ix + 1)

	// Let the new message be attached on top of the to-be-replaced message.
	c.attachGrid(ix, m.Attach())

	// Set the message into the map.
	c.messages[idKey(m.ID())] = m

	return true
}

// Around returns the message before and after the given ID, or nil if none.
func (c *GridStore) Around(id cchat.ID) (before, after GridMessage) {
	gridBefore, gridAfter := c.around(id)

	if gridBefore != nil {
		before = gridBefore.GridMessage
	}
	if gridAfter != nil {
		after = gridAfter.GridMessage
	}

	return
}

func (c *GridStore) around(id cchat.ID) (before, after *gridMessage) {
	var last *gridMessage
	var next bool

	for elem := c.messageList.Front(); elem != nil; elem = elem.Next() {
		message := elem.Value.(*gridMessage)
		if next {
			after = message
			break
		}
		if message.ID() == id {
			// The last message is the before.
			before = last
			next = true
			continue
		}

		last = message
	}
	return
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
	for elem := c.messageList.Back(); elem != nil; elem = elem.Prev() {
		gridMsg := elem.Value.(*gridMessage)
		// Ignore sending messages.
		if gridMsg.presend != nil {
			continue
		}
		if gridMsg := gridMsg.GridMessage; isMessage(gridMsg) {
			return gridMsg
		}
	}

	return nil
}

// NthMessage returns the nth message.
func (c *GridStore) NthMessage(n int) GridMessage {
	var index = 0
	for elem := c.messageList.Front(); elem != nil; elem = elem.Next() {
		if index == n {
			return elem.Value.(*gridMessage).GridMessage
		}
		index++
	}

	return nil
}

// FirstMessage returns the first message.
func (c *GridStore) FirstMessage() GridMessage {
	if c.messageList.Len() == 0 {
		return nil
	}
	// Long unwrap.
	return c.messageList.Front().Value.(*gridMessage).GridMessage
}

// LastMessage returns the latest message.
func (c *GridStore) LastMessage() GridMessage {
	if c.messageList.Len() == 0 {
		return nil
	}
	// Long unwrap.
	return c.messageList.Back().Value.(*gridMessage).GridMessage
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
	m, ok := c.messages[idKey(msgID)]
	if ok {
		return m
	}

	// Is this an existing message?
	if nonce != "" {
		// Things in this map are guaranteed to have presend != nil.
		m, ok := c.messages[nonceKey(nonce)]
		if ok {
			// Replace the nonce key with ID.
			delete(c.messages, nonceKey(nonce))
			c.messages[idKey(msgID)] = m

			// Set the right ID.
			m.presend.SetDone(msgID)
			// Destroy the presend struct.
			m.presend = nil

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
	// Append the message.
	c.messageList.PushBack(msgc)
	// Set the NONCE into the message map.
	c.messages[nonceKey(msgc.Nonce())] = msgc

	return presend
}

// Many attempts were made to have CreateMessageUnsafe return an index. That is
// unreliable. The index might be off if the message buffer is cleaned up. Don't
// rely on it.

func (c *GridStore) CreateMessageUnsafe(msg cchat.MessageCreate) {
	// Call the event handler last.
	defer c.Controller.AuthorEvent(msg.Author())

	// Attempt to update before insertion (aka upsert).
	if msgc := c.message(msg.ID(), msg.Nonce()); msgc != nil {
		msgc.UpdateAuthor(msg.Author())
		msgc.UpdateContent(msg.Content(), false)
		msgc.UpdateTimestamp(msg.Time())

		c.Controller.BindMenu(msgc.GridMessage)
		return
	}

	msgc := &gridMessage{
		GridMessage: c.Construct.NewMessage(msg),
	}
	msgTime := msg.Time()

	var index = c.messageList.Len() - 1
	var after = c.messageList.Back()

	// Iterate and compare timestamp to find where to insert a message.
	for after != nil {
		if msgTime.After(after.Value.(*gridMessage).Time()) {
			break
		}
		index--
		after = after.Prev()
	}

	// Append the message. If after is nil, then that means the message is the
	// oldest, so we add it to the front of the list.
	if after != nil {
		index++ // insert right after
		c.messageList.InsertAfter(msgc, after)
	} else {
		index = 0
		c.messageList.PushFront(msgc)
	}

	// Set the message into the grid.
	c.Grid.InsertRow(index)
	c.attachGrid(index, msgc.Attach())

	// Set the NONCE into the message map.
	c.messages[nonceKey(msgc.Nonce())] = msgc

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
func (c *GridStore) PopMessage(id cchat.ID) (msg GridMessage) {
	// Get the raw element to delete it off the list.
	elem, gridMsg, ix := c.findElement(id)
	if elem == nil {
		return nil
	}
	msg = gridMsg.GridMessage

	// Remove off of the Gtk grid.
	c.Grid.RemoveRow(ix)
	// Pop off the slice.
	c.messageList.Remove(elem)
	// Delete off the map.
	delete(c.messages, idKey(id))

	return
}

// DeleteEarliest deletes the n earliest messages. It does nothing if n is or
// less than 0.
func (c *GridStore) DeleteEarliest(n int) {
	if n <= 0 {
		return
	}

	// Since container/list nils out the next element, we can't just call Next
	// after deleting, so we have to call Next manually before Removing.
	for elem := c.messageList.Front(); elem != nil && n != 0; n-- {
		gridMsg := elem.Value.(*gridMessage)

		if id := gridMsg.ID(); id != "" {
			delete(c.messages, idKey(id))
		}
		if nonce := gridMsg.Nonce(); nonce != "" {
			delete(c.messages, nonceKey(nonce))
		}

		c.Grid.RemoveRow(0)

		next := elem.Next()
		c.messageList.Remove(elem)
		elem = next
	}
}
