package container

import (
	"strings"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

type messageKey struct {
	id    string
	nonce bool
}

func nonceKey(nonce string) messageKey { return messageKey{nonce, true} }
func idKey(id cchat.ID) messageKey     { return messageKey{id, false} }

func parseKeyFromNamer(n primitives.Namer) messageKey {
	name, err := n.GetName()
	if err != nil {
		panic("BUG: failed to get primitive name: " + err.Error())
	}

	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 {
		return messageKey{id: name}
	}

	switch parts[0] {
	case "id":
		return messageKey{id: parts[1]}
	case "nonce":
		return messageKey{id: parts[1], nonce: true}
	default:
		panic("Unknown prefix in row name " + parts[0])
	}
}

func (key messageKey) expand() (id, nonce string) {
	if key.nonce {
		return "", key.id
	}
	return key.id, ""
}

func (key messageKey) name() string {
	if key.nonce {
		return "nonce:" + key.id
	}
	return "id:" + key.id
}

// String satisfies the fmt.Stringer interface.
func (key messageKey) String() string { return key.name() }

var messageListCSS = primitives.PrepareClassCSS("message-list", `
	.message-list { background: transparent; }
`)

var fallbackAuthor = message.NewCustomAuthor("", text.Plain("self"))

type ListStore struct {
	ListBox    *gtk.ListBox
	Controller Controller

	self *message.Author

	resetMe  bool
	messages map[messageKey]*messageRow
}

func NewListStore(ctrl Controller) *ListStore {
	listBox, _ := gtk.ListBoxNew()
	listBox.SetSelectionMode(gtk.SELECTION_SINGLE)
	listBox.Show()
	messageListCSS(listBox)

	listStore := ListStore{
		ListBox:    listBox,
		Controller: ctrl,
		messages:   make(map[messageKey]*messageRow, BacklogLimit+1),
		self:       &fallbackAuthor,
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

		id := parseKeyFromNamer(r)

		msg := listStore.Message(id.expand())
		if msg == nil {
			return
		}

		selected = true
		ctrl.SelectMessage(&listStore, msg)
	})

	return &listStore
}

// Reset resets the list store.
func (c *ListStore) Reset() {
	for _, msg := range c.messages {
		destroyMsg(msg)
	}

	// Delegate removing children to the constructor.
	c.messages = make(map[messageKey]*messageRow, BacklogLimit+1)

	if c.self.ID != "" {
		c.self.Name.Stop()
	}
}

// SetSelf sets the current author to presend. If ID is empty or Namer is nil,
// then the fallback author is used instead. The given author will be stopped
// on reset.
func (c *ListStore) SetSelf(self *message.Author) {
	if self != nil {
		c.self = self
	} else {
		c.self = &fallbackAuthor
	}
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

	msgState := msg.Unwrap(false)

	// Get the current message's index.
	oldMsg, ix := c.findIndex(msgState.ID)
	if ix == -1 {
		return false
	}

	oldState := oldMsg.Unwrap(false)

	// Remove the to-be-replaced message box. We should probably reuse the row.
	c.ListBox.Remove(oldState.Row)

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.ListBox.Insert(msgState.Row, ix)

	// Set the message into the map.
	row := c.messages[idKey(msgState.ID)]
	row.MessageRow = msg
	c.bindMessage(row)

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
		id := parseKeyFromNamer(v.(primitives.Namer))
		if next {
			after = c.message(id.expand())
			return true
		}
		if !id.nonce && id.id == aroundID {
			before = last
			next = true
			return false
		}

		last = c.message(id.expand())
		return false
	})

	return
}

// findIndex searches backwards for id.
func (c *ListStore) findIndex(findID cchat.ID) (found *messageRow, index int) {
	// Faster implementation of findMessage: no map lookup is done until an ID
	// match, so the worst case is a single string hash.
	index = c.MessagesLen() - 1

	primitives.ForeachChildBackwards(c.ListBox, func(v interface{}) (stop bool) {
		id := parseKeyFromNamer(v.(primitives.Namer))
		if !id.nonce && id.id == findID {
			found = c.message(id.expand())
			return true
		}

		index--
		return index <= 0
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
		id := parseKeyFromNamer(v.(primitives.Namer))
		gridMsg := c.message(id.expand())

		// If gridMsg is actually nil, then we have bigger issues.
		if gridMsg != nil {
			// Ignore sending messages.
			if (presend || gridMsg.presend == nil) && fn(gridMsg) {
				r = gridMsg
				return true
			}
		}

		i--
		return i <= 0
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

	id := parseKeyFromNamer(v.(primitives.Namer))
	return c.message(id.expand())
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

		// This is honestly pretty dumb, but whatever.
		// TODO: make message() getter not set.
		if ok {
			// Replace the nonce key with ID.
			delete(c.messages, nonceKey(nonce))
			c.messages[idKey(msgID)] = m
			c.bindMessage(m)

			// Set the right ID.
			m.presend.SetDone(msgID)
			// Destroy the presend struct.
			m.presend = nil

			return m
		}
	}

	return nil
}

func (c *ListStore) bindMessage(msgc *messageRow) {
	state := msgc.Unwrap(false)

	// Bind the message ID to the row so we can easily do a lookup.
	key := messageKey{
		id: state.ID,
	}

	if state.Nonce != "" {
		key.id = state.Nonce
		key.nonce = true
	}

	state.Row.SetName(key.name())
	msgc.MessageRow.SetReferenceHighlighter(c)

	c.Controller.BindMenu(msgc.MessageRow)
}

func (c *ListStore) AddMessage(msg MessageRow) {
	state := msg.Unwrap(false)

	defer c.Controller.AuthorEvent(state.Author.ID)

	// Do not attempt to update before insertion (aka upsert).
	if msgc := c.message(state.ID, state.Nonce); msgc != nil {
		// This is kind of expensive, but it shouldn't really matter.
		c.SwapMessage(msg)
		return
	}

	// Iterate and compare timestamp to find where to insert a message. Note
	// that "before" is the message that will go before the to-be-inserted
	// method.
	before, index := c.findMessage(true, func(before *messageRow) bool {
		return before.Unwrap(false).Time.After(state.Time)
	})

	msgc := &messageRow{
		MessageRow: msg,
	}

	// Add the message. If before is nil, then the to-be-inserted message is the
	// earliest message, therefore we prepend it.
	if before == nil {
		index = 0
		c.ListBox.Prepend(state.Row)
	} else {
		index++ // insert right after

		// Fast path: Insert did appear a lot on profiles, so we can try and use
		// Add over Insert when we know.
		if c.MessagesLen() == index {
			c.ListBox.Add(state.Row)
		} else {
			c.ListBox.Insert(state.Row, index)
		}
	}

	// Set the ID into the message map.
	c.messages[idKey(state.ID)] = msgc

	c.bindMessage(msgc)
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
	destroyMsg(gridMsg)

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
		id := parseKeyFromNamer(v.(primitives.Namer))
		gridMsg := c.message(id.expand())

		state := gridMsg.Unwrap(false)

		if state.ID != "" {
			delete(c.messages, idKey(state.ID))
		}

		if state.Nonce != "" {
			delete(c.messages, nonceKey(state.Nonce))
		}

		destroyMsg(gridMsg)

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
	gts.ExecAsync(func() {
		state := msg.Unwrap(false)
		state.Row.GrabFocus()
		c.ListBox.DragHighlightRow(state.Row)
		gts.DoAfter(2*time.Second, c.ListBox.DragUnhighlightRow)
	})
}

func destroyMsg(row *messageRow) {
	state := row.Unwrap(true)
	state.Author.Name.Stop()
	state.Row.Destroy()
}
