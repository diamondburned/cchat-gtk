package container

import (
	"time"

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
	ListBox *gtk.ListBox

	Construct  Constructor
	Controller Controller

	resetMe bool

	messages map[messageKey]*messageRow
}

func NewListStore(ctrl Controller, constr Constructor) *ListStore {
	listBox, _ := gtk.ListBoxNew()
	listBox.SetSelectionMode(gtk.SELECTION_SINGLE)
	listBox.Show()
	messageListCSS(listBox)

	listStore := ListStore{
		ListBox:    listBox,
		Construct:  constr,
		Controller: ctrl,
		messages:   make(map[messageKey]*messageRow, BacklogLimit+1),
	}

	var selected bool

	listBox.Connect("row-selected", func(listBox *gtk.ListBox, r *gtk.ListBoxRow) {
		if r == nil || selected {
			if selected {
				listBox.UnselectAll()
				selected = false
			}
			ctrl.UnselectMessage()
			return
		}

		id, _ := r.GetName()

		msg := listStore.Message(id, "")
		if msg == nil {
			return
		}

		selected = true
		ctrl.SelectMessage(&listStore, msg)
	})

	return &listStore
}

func (c *ListStore) Reset() {
	primitives.RemoveChildren(c.ListBox)
	c.messages = make(map[messageKey]*messageRow, BacklogLimit+1)
}

func (c *ListStore) MessagesLen() int {
	return len(c.messages)
}

// Swap changes the message with the ID to the given message. This provides a
// low level API for edits that need a new Attach method.
//
// TODO: combine compact and full so they share the same attach method.
func (c *ListStore) SwapMessage(msg MessageRow) bool {
	// Unwrap msg from a *messageRow if it's not already.
	m, ok := msg.(*messageRow)
	if ok {
		msg = m.MessageRow
	}

	// Get the current message's index.
	oldMsg, ix := c.findIndex(msg.ID())
	if ix == -1 {
		return false
	}

	// Remove the to-be-replaced message box. We should probably reuse the row.
	c.ListBox.Remove(oldMsg.Row())

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.ListBox.Insert(msg.Row(), ix)

	// Set the message into the map.
	row := c.messages[idKey(msg.ID())]
	row.MessageRow = msg

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

func (c *ListStore) around(aroundID cchat.ID) (before, after *messageRow) {
	var last *messageRow
	var next bool

	primitives.ForeachChildBackwards(c.ListBox, func(v interface{}) (stop bool) {
		id := primitives.GetName(v.(primitives.Namer))
		if next {
			after = c.message(id, "")
			return true
		}
		if id == aroundID {
			before = last
			next = true
			return false
		}

		last = c.message(id, "")
		return false
	})

	return
}

// LatestMessageFrom returns the latest message with the given user ID. This is
// used for the input prompt.
func (c *ListStore) LatestMessageFrom(userID string) (msgID string, ok bool) {
	// FindMessage already looks from the latest messages.
	var msg = c.FindMessage(func(msg MessageRow) bool {
		return msg.Author().ID() == userID
	})

	if msg == nil {
		return "", false
	}

	return msg.ID(), true
}

// findIndex searches backwards for id.
func (c *ListStore) findIndex(findID cchat.ID) (found *messageRow, index int) {
	// Faster implementation of findMessage: no map lookup is done until an ID
	// match, so the worst case is a single string hash.
	index = c.MessagesLen() - 1

	primitives.ForeachChildBackwards(c.ListBox, func(v interface{}) (stop bool) {
		id := primitives.GetName(v.(primitives.Namer))
		if id == findID {
			found = c.message(findID, "")
			return true
		}

		index--
		return index == 0
	})

	// Preserve old behavior.
	if found == nil {
		index = -1
	}

	return
}

func (c *ListStore) findMessage(presend bool, fn func(*messageRow) bool) (*messageRow, int) {
	var r *messageRow
	var i = c.MessagesLen() - 1

	primitives.ForeachChildBackwards(c.ListBox, func(v interface{}) (stop bool) {
		id := primitives.GetName(v.(primitives.Namer))
		gridMsg := c.message(id, "")

		// If gridMsg is actually nil, then we have bigger issues.
		if gridMsg != nil {
			// Ignore sending messages.
			if (presend || gridMsg.presend == nil) && fn(gridMsg) {
				r = gridMsg
				return true
			}
		}

		i--
		return false
	})

	// Preserve old behavior.
	if r == nil {
		i = -1
	}

	return r, i
}

// FindMessage iterates backwards and returns the message if isMessage() returns
// true on that message. It does not search presend messages.
func (c *ListStore) FindMessage(isMessage func(MessageRow) bool) MessageRow {
	msg, _ := c.findMessage(false, func(row *messageRow) bool {
		return isMessage(row.MessageRow)
	})
	return unwrapRow(msg)
}

func (c *ListStore) nthMessage(n int) *messageRow {
	v := primitives.NthChild(c.ListBox, n)
	if v == nil {
		return nil
	}

	id := primitives.GetName(v.(primitives.Namer))
	return c.message(id, "")
}

// NthMessage returns the nth message.
func (c *ListStore) NthMessage(n int) MessageRow {
	return unwrapRow(c.nthMessage(n))
}

// FirstMessage returns the first message.
func (c *ListStore) FirstMessage() MessageRow {
	return c.NthMessage(0)
}

// LastMessage returns the latest message.
func (c *ListStore) LastMessage() MessageRow {
	return c.NthMessage(c.MessagesLen() - 1)
}

// Message finds the message state in the container. It is not thread-safe. This
// exists for backwards compatibility.
func (c *ListStore) Message(msgID cchat.ID, nonce string) MessageRow {
	return unwrapRow(c.message(msgID, nonce))
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
	before := c.LastMessage()
	presend := c.Construct.NewPresendMessage(msg, before)

	msgc := &messageRow{
		MessageRow: presend,
		presend:    presend,
	}

	// Set the message into the list.
	c.ListBox.Insert(msgc.Row(), c.MessagesLen())
	// Set the NONCE into the message map.
	c.messages[nonceKey(msgc.Nonce())] = msgc

	return presend
}

func (c *ListStore) bindMessage(msgc *messageRow) {
	// Bind the message ID to the row so we can easily do a lookup.
	msgc.Row().SetName(msgc.ID())
	msgc.SetReferenceHighlighter(c)
	c.Controller.BindMenu(msgc.MessageRow)
}

// Many attempts were made to have CreateMessageUnsafe return an index. That is
// unreliable. The index might be off if the message buffer is cleaned up. Don't
// rely on it.

func (c *ListStore) CreateMessageUnsafe(msg cchat.MessageCreate) MessageRow {
	// Call the event handler last.
	defer c.Controller.AuthorEvent(msg.Author())

	// Do not attempt to update before insertion (aka upsert).
	if msgc := c.message(msg.ID(), msg.Nonce()); msgc != nil {
		msgc.UpdateAuthor(msg.Author())
		msgc.UpdateContent(msg.Content(), false)
		msgc.UpdateTimestamp(msg.Time())

		c.bindMessage(msgc)
		return msgc.MessageRow
	}

	msgTime := msg.Time()

	// Iterate and compare timestamp to find where to insert a message. Note
	// that "before" is the message that will go before the to-be-inserted
	// method.
	before, index := c.findMessage(true, func(before *messageRow) bool {
		return msgTime.After(before.Time())
	})

	msgc := &messageRow{
		MessageRow: c.Construct.NewMessage(msg, unwrapRow(before)),
	}

	// Add the message. If before is nil, then the to-be-inserted message is the
	// earliest message, therefore we prepend it.
	if before == nil {
		index = 0
		c.ListBox.Prepend(msgc.Row())
	} else {
		index++ // insert right after

		// Fast path: Insert did appear a lot on profiles, so we can try and use
		// Add over Insert when we know.
		if c.MessagesLen() == index {
			c.ListBox.Add(msgc.Row())
		} else {
			c.ListBox.Insert(msgc.Row(), index)
		}
	}

	// Set the ID into the message map.
	c.messages[idKey(msgc.ID())] = msgc

	c.bindMessage(msgc)

	return msgc.MessageRow
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
	gridMsg, _ := c.findIndex(id)
	if gridMsg == nil {
		return nil
	}
	msg = gridMsg.MessageRow

	// Remove off of the Gtk grid.
	gridMsg.Row().Destroy()
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
	primitives.ForeachChild(c.ListBox, func(v interface{}) (stop bool) {
		id := primitives.GetName(v.(primitives.Namer))
		gridMsg := c.message(id, "")

		if id := gridMsg.ID(); id != "" {
			delete(c.messages, idKey(id))
		}
		if nonce := gridMsg.Nonce(); nonce != "" {
			delete(c.messages, nonceKey(nonce))
		}

		gridMsg.Row().Destroy()

		n--
		return n == 0
	})
}

func (c *ListStore) HighlightReference(ref markup.ReferenceSegment) {
	msg := c.message(ref.MessageID(), "")
	if msg != nil {
		c.Highlight(msg)
	}
}

func (c *ListStore) Highlight(msg MessageRow) {
	gts.ExecLater(func() {
		row := msg.Row()
		row.GrabFocus()
		c.ListBox.DragHighlightRow(row)
		gts.DoAfter(2*time.Second, c.ListBox.DragUnhighlightRow)
	})
}
