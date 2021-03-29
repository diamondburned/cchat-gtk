package input

import (
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const shiftMask = uint(gdk.SHIFT_MASK)
const cntrlMask = uint(gdk.CONTROL_MASK)

func bithas(bit, has uint) bool {
	return bit&has == has
}

func convEvent(ev *gdk.Event) (key, mask uint) {
	var keyEvent = gdk.EventKeyNewFromEvent(ev)
	return keyEvent.KeyVal(), keyEvent.State()
}

// connects to key-press-event
func (f *Field) keyDown(tv *gtk.TextView, ev *gdk.Event) bool {
	var key, mask = convEvent(ev)

	switch {
	// If Enter is pressed.
	case key == gdk.KEY_Return:
		// If Shift is being held, insert a new line.
		if bithas(mask, shiftMask) {
			f.buffer.InsertAtCursor("\n")
			return true
		}

		// Else, send the message.
		f.sendInput()
		return true

	// If Arrow Up is pressed, then we might want to edit the latest message if
	// any.
	case key == gdk.KEY_Up:
		// Do we have input? If we do, then we shouldn't touch it.
		if f.textLen() > 0 {
			return false
		}

		// Try and find the latest message ID that is ours.
		msgr := f.ctrl.LatestMessageFrom(f.UserID)
		if msgr == nil {
			// No messages found, so we can passthrough normally.
			return false
		}

		id := msgr.Unwrap(false).ID

		// If we don't support message editing, then passthrough events.
		if !f.Editable(id) {
			return false
		}

		// Start editing.
		f.StartEditing(id)

		// TODO: add a visible indicator to indicate a message being edited.

		// Take the event.
		return true

	// Clear text when the Escape key is pressed.
	case key == gdk.KEY_Escape:
		f.clearText()

	// Ctrl+V is paste.
	case key == gdk.KEY_v && bithas(mask, cntrlMask):
		// As this pasting is for image attachments, don't accept it if iwe
		// don't allow attachments.
		if !f.upload {
			return false
		}

		// TODO: make this asynchronous.

		// Is there an image in the clipboard?
		if !gts.Clipboard.WaitIsImageAvailable() {
			return false
		}

		gts.Async(func() (func(), error) {
			p, err := gts.Clipboard.WaitForImage()
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get image from clipboard")
			}

			return func() { f.Attachments.AddPixbuf(p) }, nil
		})

		return true
	}

	// If the server supports typing indication, then announce that we are
	// typing with a proper rate limit.
	if f.typing != nil {
		// Get the current time; if the next timestamp is before now, then that
		// means it's time for us to update it and send a typing indication.
		if now := time.Now(); f.lastTyped.Add(f.typerDura).Before(now) {
			// Update.
			f.lastTyped = now
			// Send asynchronously.
			go func() {
				if err := f.typing.Typing(); err != nil {
					log.Error(errors.Wrap(err, "Failed to announce typing"))
				}
			}()
		}
	}

	// Passthrough.
	return false
}
