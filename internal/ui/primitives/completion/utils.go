package completion

import (
	"unicode"

	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

var popoverCSS = primitives.PrepareCSS(`
	popover {
		border-radius: 0;
	}
`)

const (
	MinPopoverWidth = 300
)

func NewPopover(relto gtk.IWidget) *gtk.Popover {
	p, _ := gtk.PopoverNew(relto)
	p.SetSizeRequest(MinPopoverWidth, -1)
	p.SetModal(false)
	p.SetPosition(gtk.POS_TOP)
	primitives.AttachCSS(p, popoverCSS)
	return p
}

type KeyDownHandlerFn = func(gtk.IWidget, *gdk.Event) bool

func KeyDownHandler(l *gtk.ListBox, focus func()) KeyDownHandlerFn {
	return func(w gtk.IWidget, ev *gdk.Event) bool {
		// Do we have any entries? If not, don't bother.
		var length = primitives.ChildrenLen(l)
		if length == 0 {
			// passthrough.
			return false
		}

		var evKey = gdk.EventKeyNewFromEvent(ev)
		var key = evKey.KeyVal()

		switch key {
		// Did we press an arrow key?
		case gdk.KEY_Up, gdk.KEY_Down:
			// Yes, start moving the list up and down.
			i := l.GetSelectedRow().GetIndex()

			switch key {
			case gdk.KEY_Up:
				if i--; i < 0 {
					i = length - 1
				}
			case gdk.KEY_Down:
				if i++; i >= length {
					i = 0
				}
			}

			row := l.GetRowAtIndex(i)
			row.GrabFocus()  // scroll
			l.SelectRow(row) // select
			focus()          // unfocus

		// Did we press the Enter or Tab key?
		case gdk.KEY_Return, gdk.KEY_Tab:
			// Activate the current row.
			l.GetSelectedRow().Activate()
			focus()

		default:
			// passthrough events if none matches.
			return false
		}

		return true
	}
}

func SwapWord(b *gtk.TextBuffer, word string, offset int64) {
	// Get iter for word replacing.
	start, end := GetWordIters(b, offset)
	b.Delete(start, end)
	b.Insert(start, word+" ")
}

func CursorRect(i *gtk.TextView) gdk.Rectangle {
	r, _ := i.GetCursorLocations(nil)
	x, _ := i.BufferToWindowCoords(gtk.TEXT_WINDOW_WIDGET, r.GetX(), r.GetY())
	r.SetX(x)
	r.SetY(0)
	return *r
}

func State(buf *gtk.TextBuffer) (text string, offset int64, blank bool) {
	// obtain current state
	mark := buf.GetInsert()
	iter := buf.GetIterAtMark(mark)

	// obtain the input string and the current cursor position
	start, end := buf.GetBounds()

	text, _ = buf.GetText(start, end, true)
	offset = int64(iter.GetOffset())

	// We need the rune before the cursor.
	iter.BackwardChar()
	char := iter.GetChar()

	// Treat NULs as blanks.
	blank = unicode.IsSpace(char) || char == '\x00'

	return
}

const searchFlags = 0 |
	gtk.TEXT_SEARCH_TEXT_ONLY |
	gtk.TEXT_SEARCH_VISIBLE_ONLY

func GetWordIters(buf *gtk.TextBuffer, offset int64) (start, end *gtk.TextIter) {
	iter := buf.GetIterAtOffset(int(offset))

	var ok bool

	// Seek backwards for space or start-of-line:
	_, start, ok = iter.BackwardSearch(" ", searchFlags, nil)
	if !ok {
		start = buf.GetStartIter()
	}

	// Seek forwards for space or end-of-line:
	_, end, ok = iter.ForwardSearch(" ", searchFlags, nil)
	if !ok {
		end = buf.GetEndIter()
	}

	return
}
