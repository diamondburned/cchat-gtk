package session

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/drag"
	"github.com/gotk3/gotk3/gtk"
)

type ServiceController interface {
	SessionSelected(*Row)
	AuthenticateSession()
}

type List struct {
	*gtk.ListBox
	// This map isn't ordered, as we rely on the order that the widget was added
	// into the ListBox.
	sessions map[string]*Row

	svcctrl ServiceController
}

var listCSS = primitives.PrepareClassCSS("session-list", `
	.session-list {
		border-radius: 0 0 14px 14px;
		background-color: mix(@theme_bg_color, @theme_fg_color, 0.1);
		box-shadow:
			inset 0  10px 2px -10px alpha(#121212, 0.6),
			inset 0 -10px 2px -10px alpha(#121212, 0.6);
	}
`)

func NewList(svcctrl ServiceController) *List {
	list, _ := gtk.ListBoxNew()
	list.Add(NewAddButton()) // add button to LAST; keep it LAST.
	list.Show()
	listCSS(list)

	// We can't do browse for the selection mode, as we need UnselectAll to
	// work.
	list.SetSelectionMode(gtk.SELECTION_SINGLE)

	sl := &List{
		ListBox:  list,
		sessions: map[string]*Row{},
		svcctrl:  svcctrl,
	}

	list.Connect("row-activated", func(l *gtk.ListBox, r *gtk.ListBoxRow) {
		switch i, length := r.GetIndex(), len(sl.sessions); {
		case i < 0:
			return // lulwut

		// If the selection IS the last button.
		case i == length:
			svcctrl.AuthenticateSession()

		// If the selection is within range and is not the last button.
		case i < length:
			if row, ok := sl.sessions[primitives.GetName(r)]; ok {
				row.Activate()
			}
		}
	})

	return sl
}

func (sl *List) Sessions() []*Row {
	// We already know the size beforehand. Allocate it wisely.
	var rows = make([]*Row, 0, len(sl.sessions))

	// Loop over widget children.
	primitives.EachChildren(sl.ListBox, func(i int, v interface{}) bool {
		var id = primitives.GetName(v.(primitives.Namer))

		if row, ok := sl.sessions[id]; ok {
			rows = append(rows, row)
		}

		return false
	})

	return rows
}

// Session returns the session row with the given ID. A nil Row is returned if
// none is found.
func (sl *List) Session(id string) *Row {
	row, _ := sl.sessions[id]
	return row
}

// AddSessionRow adds the given row as a session into the list.
func (sl *List) AddSessionRow(id string, row *Row) {
	// !!! IMPORTANT !!! Guarantee that there is NO collision.
	if _, ok := sl.sessions[id]; ok {
		panic("BUG: Duplicate session; AddSessionRow caller did not check Session.")
	}

	// Insert the row RIGHT BEFORE the add button.
	sl.ListBox.Insert(row, len(sl.sessions))
	// Set the map, which increases the length by 1.
	sl.sessions[id] = row

	// Assert that a name can be obtained.
	namer := primitives.Namer(row)
	namer.SetName(id) // set ID here, get it in Move
}

func (sl *List) RemoveSessionRow(sessionID string) bool {
	r, ok := sl.sessions[sessionID]
	if ok {
		delete(sl.sessions, sessionID)
		sl.ListBox.Remove(r)
	}
	return ok
}

// MoveSession moves sessions around. This function must not touch the add
// button.
func (sl *List) MoveSession(targetID, movingID string) {
	// Get the widget of the row that is moving.
	var moving, ok = sl.sessions[movingID]
	if !ok {
		return // sometimes movingID might come from other services
	}

	// Find the current position of the row that we're moving the other one
	// underneath of.
	var rowix = drag.Find(sl.ListBox, targetID)

	// Reorder the box.
	sl.ListBox.Remove(moving)
	sl.ListBox.Insert(moving, rowix)
}

func (sl *List) UnselectAll() {
	sl.ListBox.UnselectAll()
}
