package input

import (
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
		id, ok := f.ctrl.LatestMessageFrom(f.UserID)
		if !ok {
			// No messages found, so we can passthrough normally.
			return false
		}

		// If we don't support message editing, then passthrough events.
		if !f.Editable(id) {
			return false
		}

		// Start editing.
		f.StartEditing(id)

		// TODO: add a visible indicator to indicate a message being edited.

		// Take the event.
		return true

	// There are multiple things to do here when we press the Escape key.
	case key == gdk.KEY_Escape:
		// First, we'd want to cancel editing if we have one.
		if f.editingID != "" {
			return f.StopEditing() // always returns true
		}

		// Second... Nothing yet?

	// Ctrl+V is paste.
	case key == gdk.KEY_v && bithas(mask, cntrlMask):
		// Is there an image in the clipboard?
		if !gts.Clipboard.WaitIsImageAvailable() {
			// No.
			return false
		}
		// Yes.

		p, err := gts.Clipboard.WaitForImage()
		if err != nil {
			log.Error(errors.Wrap(err, "Failed to get image from clipboard"))
			return true // interrupt as technically valid
		}

		if err := f.Attachments.AddPixbuf(p); err != nil {
			log.Error(errors.Wrap(err, "Failed to add image to attachment list"))
			return true
		}
	}

	// Passthrough.
	return false
}
