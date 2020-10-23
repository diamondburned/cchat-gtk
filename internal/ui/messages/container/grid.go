package container

import (
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

	store *messageStore
	// messages   map[string]*gridMessage
	// messageIDs []string // ids or nonces
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
		store:      newMessageStore(),
	}
}

func (c *GridStore) MessagesLen() int {
	return c.store.Len()
}

func (c *GridStore) attachGrid(row int, widgets []gtk.IWidget) {
	// // Cover a special case with attaching to the 0th row.
	// switch row {
	// case 0:
	// 	c.Grid.InsertRow(0)
	// case c.MessagesLen() - 1:
	// 	row++ // ensure this doesn't try to write to the last message.
	// 	c.Grid.InsertRow(row)
	// }

	c.Grid.InsertRow(row)

	log.Println("Inserted row", row, "; length is", c.MessagesLen())

	for i, w := range widgets {
		c.Grid.Attach(w, i, row, 1, 1)
	}
}

type CoordinateTranslator interface {
	TranslateCoordinates(dest gtk.IWidget, srcX int, srcY int) (destX int, destY int, e error)
}

var _ CoordinateTranslator = (*gtk.Widget)(nil)

func (c *GridStore) TranslateCoordinates(parent gtk.IWidget, msg GridMessage) (y int) {
	m := c.store.Message(msg.ID(), "")
	if m == nil {
		return 0
	}

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
	// Wrap msg inside a *gridMessage if it's not already.
	mg, ok := msg.(*gridMessage)
	if !ok {
		mg = &gridMessage{GridMessage: msg}
	}

	// Get the current message's index.
	var ix = c.store.SwapMessage(mg)
	if ix == -1 {
		return false
	}

	// Add a row at index. The actual row we want to delete will be shifted
	// downwards.
	c.Grid.InsertRow(ix)

	// Let the new message be attached on top of the to-be-replaced message.
	c.attachGrid(ix, mg.Attach())

	// Delete the to-be-replaced message, which we have shifted downwards
	// earlier, so we add 1.
	c.Grid.RemoveRow(ix + 1)

	return true
}

// Before returns the message before the given ID, or nil if none.
func (c *GridStore) Before(id string) GridMessage {
	return c.store.MessageBefore(id)
}

// After returns the message after the given ID, or nil if none.
func (c *GridStore) After(id string) GridMessage {
	return c.store.MessageAfter(id)
}

// LatestMessageFrom returns the latest message with the given user ID. This is
// used for the input prompt.
func (c *GridStore) LatestMessageFrom(userID string) (msgID string, ok bool) {
	msg := c.store.LastMessageFrom(userID)
	if msg == nil {
		// "Backwards-compatibility is repeating the mistakes of yesterday,
		// today."
		return "", false
	}
	return msg.ID(), true
}

// FindMessage iterates backwards and returns the message if isMessage() returns
// true on that message.
func (c *GridStore) FindMessage(isMessage func(msg GridMessage) bool) GridMessage {
	return c.store.FindMessage(isMessage)
}

// NthMessage returns the nth message.
func (c *GridStore) NthMessage(n int) GridMessage {
	return c.store.NthMessage(n).unwrap()
}

// FirstMessage returns the first message.
func (c *GridStore) FirstMessage() GridMessage {
	return c.store.FirstMessage().unwrap()
}

// LastMessage returns the latest message.
func (c *GridStore) LastMessage() GridMessage {
	return c.store.LastMessage().unwrap()
}

// Message finds the message state in the container. It is not thread-safe. This
// exists for backwards compatibility.
func (c *GridStore) Message(msgID cchat.ID, nonce string) GridMessage {
	return c.store.Message(msgID, nonce).unwrap()
}

// AddPresendMessage inserts an input.PresendMessage into the container and
// returning a wrapped widget interface.
func (c *GridStore) AddPresendMessage(msg input.PresendMessage) PresendGridMessage {
	presend := c.Construct.NewPresendMessage(msg)

	msgc := &gridMessage{
		GridMessage: presend,
		presend:     presend,
	}

	// Crash and burn if -1 is returned.
	ix := c.store.InsertMessage(msgc)
	if ix == -1 {
		panic("BUG: -1 returned from store.InsertMessage")
	}

	// Set the message into the grid.
	c.attachGrid(ix, msgc.Attach())

	return presend
}

// CreateMessageUnsafe adds msg into the message view. It returns -1 if the
// message was "upserted," that is if it's updated instead of inserted.
func (c *GridStore) CreateMessageUnsafe(msg cchat.MessageCreate) int {
	// Call the event handler last.
	defer c.Controller.OnAuthorEvent(msg.Author())

	// Attempt to update before insertion (aka upsert).
	if msgc := c.Message(msg.ID(), msg.Nonce()); msgc != nil {
		msgc.UpdateAuthor(msg.Author())
		msgc.UpdateContent(msg.Content(), false)
		msgc.UpdateTimestamp(msg.Time())

		c.Controller.BindMenu(msgc)
		return -1
	}

	msgc := &gridMessage{
		GridMessage: c.Construct.NewMessage(msg),
	}

	// Crash and burn if -1 is returned.
	ix := c.store.InsertMessage(msgc)
	if ix == -1 {
		panic("BUG: -1 returned from store.InsertMessage")
	}

	// Set the message into the grid.
	c.attachGrid(ix, msgc.Attach())
	c.Controller.BindMenu(msgc)

	return ix
}

func (c *GridStore) UpdateMessageUnsafe(msg cchat.MessageUpdate) {
	// Call the event handler last.
	defer c.Controller.OnAuthorEvent(msg.Author())

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
	c.store.DeleteMessage(msg.ID())
}

// PopMessage deletes a message off of the list and return the deleted message.
func (c *GridStore) PopMessage(id string) GridMessage {
	msg, ix := c.store.PopMessage(id)
	if msg == nil {
		return nil
	}

	// Remove off of the Gtk grid.
	c.Grid.RemoveRow(ix)

	return msg.GridMessage
}

func (c *GridStore) PopEarliestMessages(n int) {
	poppedIxs := c.store.PopEarliestMessages(n)
	if poppedIxs == 0 {
		return
	}
	// Get the count of messages after deletion. We can then gradually decrement
	// poppedN to get the deleted message indices.
	for poppedIxs > 0 {
		c.Grid.RemoveRow(poppedIxs)
		poppedIxs--
	}
}
