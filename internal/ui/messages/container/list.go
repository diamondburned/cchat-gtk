package container

import (
	"container/list"
	"log"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/gotk3/gotk3/gtk"
)

type messageKey struct {
	id    string
	nonce bool
}

func nonceKey(nonce string) messageKey { return messageKey{nonce, true} }
func idKey(id cchat.ID) messageKey     { return messageKey{id, false} }

var messageListCSS = primitives.PrepareClassCSS("message-list", `
	.message-list { background: transparent; }
`)

type ListStore struct {
	*gtk.ListBox

	Construct  Constructor
	Controller Controller

	resetMe bool

	messages    map[messageKey]*messageRow
	messageList *list.List
}

func NewListStore(constr Constructor, ctrl Controller) *ListStore {
	listBox, _ := gtk.ListBoxNew()
	listBox.SetSelectionMode(gtk.SELECTION_NONE)
	listBox.Show()
	messageListCSS(listBox)

	return &ListStore{
		ListBox:     listBox,
		Construct:   constr,
		Controller:  ctrl,
		messages:    make(map[messageKey]*messageRow, BacklogLimit+1),
		messageList: list.New(),
	}
}

func (c *ListStore) Reset() {
	primitives.RemoveChildren(c.ListBox)
	c.messages = make(map[messageKey]*messageRow, BacklogLimit+1)
	c.messageList = list.New()
}

func (c *ListStore) MessagesLen() int {
	return c.messageList.Len()
}

func (c *ListStore) findElement(id cchat.ID) (*list.Element, *messageRow, int) {
	var index = c.messageList.Len() - 1
	for elem := c.messageList.Back(); elem != nil; elem = elem.Prev() {
		if gridMsg := elem.Value.(*messageRow); gridMsg.ID() == id {
			return elem, gridMsg, index
		}
		index--
	}
	return nil, nil, -1
}

// findIndex searches backwards for id.
func (c *ListStore) findIndex(id cchat.ID) (*messageRow, int) {
	_, gridMsg, ix := c.findElement(id)
	return gridMsg, ix
}

// Swap changes the message with the ID to the given message. This provides a
// low level API for edits that need a new Attach method.
//
// TODO: combine compact and full so they share the same attach method.
func (c *ListStore) SwapMessage(msg MessageRow) bool {
	// Wrap msg inside a *messageRow if it's not already.
	m, ok := msg.(*messageRow)
	if !ok {
		m = &messageRow{MessageRow: msg}
	}

	// Get the current message's index.
	oldMsg, ix := c.findIndex(msg.ID())
	if ix == -1 {
		return false
	}

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.ListBox.Insert(m.Row(), ix)

	// Delete the to-be-replaced message.
	oldMsg.Row().Destroy()

	// Set the message into the map.
	row := c.messages[idKey(m.ID())]
	*row = *m

	return true
}

// Around returns the message before and after the given ID, or nil if none.
func (c *ListStore) Around(id cchat.ID) (before, after MessageRow) {
	gridBefore, gridAfter := c.around(id)

	if gridBefore != nil {
		before = gridBefore.MessageRow
	}
	if gridAfter != nil {
		after = gridAfter.MessageRow
	}

	return
}

func (c *ListStore) around(id cchat.ID) (before, after *messageRow) {
	var last *messageRow
	var next bool

	for elem := c.messageList.Front(); elem != nil; elem = elem.Next() {
		message := elem.Value.(*messageRow)
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
func (c *ListStore) LatestMessageFrom(userID string) (msgID string, ok bool) {
	// FindMessage already looks from the latest messages.
	var msg = c.FindMessage(func(msg MessageRow) bool {
		return msg.AuthorID() == userID
	})

	if msg == nil {
		return "", false
	}

	return msg.ID(), true
}

// FindMessage iterates backwards and returns the message if isMessage() returns
// true on that message.
func (c *ListStore) FindMessage(isMessage func(msg MessageRow) bool) MessageRow {
	for elem := c.messageList.Back(); elem != nil; elem = elem.Prev() {
		gridMsg := elem.Value.(*messageRow)
		// Ignore sending messages.
		if gridMsg.presend != nil {
			continue
		}
		if gridMsg := gridMsg.MessageRow; isMessage(gridMsg) {
			return gridMsg
		}
	}

	return nil
}

// NthMessage returns the nth message.
func (c *ListStore) NthMessage(n int) MessageRow {
	var index = 0
	for elem := c.messageList.Front(); elem != nil; elem = elem.Next() {
		if index == n {
			return elem.Value.(*messageRow).MessageRow
		}
		index++
	}

	return nil
}

// FirstMessage returns the first message.
func (c *ListStore) FirstMessage() MessageRow {
	if c.messageList.Len() == 0 {
		return nil
	}
	// Long unwrap.
	return c.messageList.Front().Value.(*messageRow).MessageRow
}

// LastMessage returns the latest message.
func (c *ListStore) LastMessage() MessageRow {
	if c.messageList.Len() == 0 {
		return nil
	}
	// Long unwrap.
	return c.messageList.Back().Value.(*messageRow).MessageRow
}

// Message finds the message state in the container. It is not thread-safe. This
// exists for backwards compatibility.
func (c *ListStore) Message(msgID cchat.ID, nonce string) MessageRow {
	if m := c.message(msgID, nonce); m != nil {
		return m.MessageRow
	}
	return nil
}

func (c *ListStore) message(msgID cchat.ID, nonce string) *messageRow {
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
func (c *ListStore) AddPresendMessage(msg input.PresendMessage) PresendMessageRow {
	presend := c.Construct.NewPresendMessage(msg)

	msgc := &messageRow{
		MessageRow: presend,
		presend:    presend,
	}

	// Set the message into the list.
	c.ListBox.Insert(msgc.Row(), c.MessagesLen())
	// Append the message.
	c.messageList.PushBack(msgc)
	// Set the NONCE into the message map.
	c.messages[nonceKey(msgc.Nonce())] = msgc

	return presend
}

func (c *ListStore) bindMessage(msgc *messageRow) {
	msgc.SetReferenceHighlighter(c)
	c.Controller.BindMenu(msgc.MessageRow)
}

// Many attempts were made to have CreateMessageUnsafe return an index. That is
// unreliable. The index might be off if the message buffer is cleaned up. Don't
// rely on it.

func (c *ListStore) CreateMessageUnsafe(msg cchat.MessageCreate) {
	// Call the event handler last.
	defer c.Controller.AuthorEvent(msg.Author())

	// Do not attempt to update before insertion (aka upsert).
	if msgc := c.message(msg.ID(), msg.Nonce()); msgc != nil {
		msgc.UpdateAuthor(msg.Author())
		msgc.UpdateContent(msg.Content(), false)
		msgc.UpdateTimestamp(msg.Time())

		c.bindMessage(msgc)
		return
	}

	msgc := &messageRow{
		MessageRow: c.Construct.NewMessage(msg),
	}
	msgTime := msg.Time()

	var index = c.messageList.Len() - 1
	var after = c.messageList.Back()

	// Iterate and compare timestamp to find where to insert a message.
	for after != nil {
		if msgTime.After(after.Value.(*messageRow).Time()) {
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
	c.ListBox.Insert(msgc.Row(), index)

	// Set the ID into the message map.
	c.messages[idKey(msgc.ID())] = msgc

	c.bindMessage(msgc)
}

func (c *ListStore) UpdateMessageUnsafe(msg cchat.MessageUpdate) {
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

func (c *ListStore) DeleteMessageUnsafe(msg cchat.MessageDelete) {
	c.PopMessage(msg.ID())
}

// PopMessage deletes a message off of the list and return the deleted message.
func (c *ListStore) PopMessage(id cchat.ID) (msg MessageRow) {
	// Get the raw element to delete it off the list.
	elem, gridMsg, _ := c.findElement(id)
	if elem == nil {
		return nil
	}
	msg = gridMsg.MessageRow

	// Remove off of the Gtk grid.
	gridMsg.Row().Destroy()
	// Pop off the slice.
	c.messageList.Remove(elem)
	// Delete off the map.
	delete(c.messages, idKey(id))

	return
}

// DeleteEarliest deletes the n earliest messages. It does nothing if n is or
// less than 0.
func (c *ListStore) DeleteEarliest(n int) {
	if n <= 0 {
		return
	}

	// Since container/list nils out the next element, we can't just call Next
	// after deleting, so we have to call Next manually before Removing.
	for elem := c.messageList.Front(); elem != nil && n != 0; n-- {
		gridMsg := elem.Value.(*messageRow)

		if id := gridMsg.ID(); id != "" {
			delete(c.messages, idKey(id))
		}
		if nonce := gridMsg.Nonce(); nonce != "" {
			delete(c.messages, nonceKey(nonce))
		}

		gridMsg.Row().Destroy()

		next := elem.Next()
		c.messageList.Remove(elem)
		elem = next
	}
}

func (c *ListStore) HighlightReference(ref markup.ReferenceSegment) {
	msg := c.message(ref.MessageID(), "")
	log.Println("Highlighting", ref.MessageID())
	if msg != nil {
		c.Highlight(msg)
	}
}

func (c *ListStore) Highlight(msg MessageRow) {
	gts.ExecLater(func() {
		row := msg.Row()
		row.GrabFocus()
		c.ListBox.DragHighlightRow(row)
	})
}

func (c *ListStore) Unhighlight() {
	c.ListBox.DragUnhighlightRow()
}
