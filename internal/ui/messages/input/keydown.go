package input

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
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

	switch key {
	// If Enter is pressed.
	case gdk.KEY_Return:
		// If Shift is being held, insert a new line.
		if bithas(mask, shiftMask) {
			f.buffer.InsertAtCursor("\n")
			return true
		}

		// Else, send the message.
		f.SendInput()
		return true
	}

	// Passthrough.
	return false
}
