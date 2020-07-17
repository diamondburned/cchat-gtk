package drag

import (
	"net/url"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func NewTargetEntry(target string, f gtk.TargetFlags, info uint) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, f, info)
	return *e
}

// Find searches the given container for the draggable widget with the given
// name.
func Find(w primitives.Container, id string) int {
	var index = -1 // not found default

	primitives.EachChildren(w, func(i int, v interface{}) bool {
		if primitives.GetName(v.(primitives.Namer)) == id {
			index = i
			return true
		}

		return false
	})

	return index
}

type MainDraggable interface {
	ID() string
	SetName(string)
	SetSensitive(bool)

	gtk.IWidget
	Draggable
}

type Draggable interface {
	DragSourceSet(gdk.ModifierType, []gtk.TargetEntry, gdk.DragAction)
	DragDestSet(gtk.DestDefaults, []gtk.TargetEntry, gdk.DragAction)

	primitives.Connector
}

func BindFileDest(dg Draggable, file func(path []string)) {
	var dragEntries = []gtk.TargetEntry{
		NewTargetEntry("text/uri-list", gtk.TARGET_OTHER_APP, 1),
	}

	dg.DragDestSet(gtk.DEST_DEFAULT_ALL, dragEntries, gdk.ACTION_COPY)
	dg.Connect("drag-data-received",
		func(_ gtk.IWidget, ctx *gdk.DragContext, x, y uint, data *gtk.SelectionData) {
			// Get the files in form of line-delimited URIs
			var uris = strings.Fields(string(data.GetData()))

			// Create a path slice that we decode URIs into.
			var paths = uris[:0]

			// Decode the URIs.
			for _, uri := range uris {
				u, err := url.Parse(uri)
				if err != nil {
					log.Error(errors.Wrapf(err, "Failed parsing URI %q", uri))
					continue
				}
				paths = append(paths, u.Path)
			}

			file(paths)
		},
	)
}

// Swapper is the type for a swap function.
type Swapper = func(targetID, movingID string)

// BindDraggable binds the draggable widget and make it drag-and-droppable. The
// parent MUST have its own state of children and MUST NOT rely on its container
// states.
//
// This function can take additional draggers, which will override the main
// draggable and will be the only widgets that can be dragged away. The source
// ID will be taken from the main draggable.
func BindDraggable(dg MainDraggable, icon string, fn Swapper, draggers ...Draggable) {
	var atom = "data_" + icon
	var dragAtom = gdk.GdkAtomIntern(atom, false)
	var dragEntries = []gtk.TargetEntry{
		NewTargetEntry(atom, gtk.TARGET_SAME_APP, 0),
	}

	// Set the ID for Find().
	dg.SetName(dg.ID())

	// Make closures function so we can use twice.
	srcSet := func(dragger Draggable) {
		// Drag source so you can drag the button away.
		dragger.DragSourceSet(gdk.BUTTON1_MASK, dragEntries, gdk.ACTION_MOVE)

		dragger.Connect("drag-data-get",
			func(_ gtk.IWidget, ctx *gdk.DragContext, data *gtk.SelectionData) {
				// Set the index-in-bytes.
				data.SetData(dragAtom, []byte(dg.ID()))
			},
		)

		dragger.Connect("drag-begin",
			func(_ gtk.IWidget, ctx *gdk.DragContext) {
				gtk.DragSetIconName(ctx, icon, 0, 0)
				dg.SetSensitive(false)
			},
		)

		dragger.Connect("drag-end",
			func() {
				dg.SetSensitive(true)
			},
		)
	}
	dstSet := func(dragger Draggable) {
		// Drag destination so you can drag the button here.
		dragger.DragDestSet(gtk.DEST_DEFAULT_ALL, dragEntries, gdk.ACTION_MOVE)

		dragger.Connect("drag-data-received",
			func(_ gtk.IWidget, ctx *gdk.DragContext, x, y uint, data *gtk.SelectionData) {
				// Receive the incoming row's ID and call MoveSession.
				fn(dg.ID(), string(data.GetData()))
			},
		)
	}

	// If we have no extra draggers given, then the MainDraggable should also be
	// a source.
	if len(draggers) == 0 {
		srcSet(dg)
	} else {
		// Else, set drag sources only on those extra draggables.
		for _, dragger := range draggers {
			srcSet(dragger)
			dstSet(dragger)
		}
	}

	// Make MainDraggable a drag destination as well.
	dstSet(dg)
}
