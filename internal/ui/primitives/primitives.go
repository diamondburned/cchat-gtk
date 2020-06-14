package primitives

import (
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type Namer interface {
	SetName(string)
	GetName() (string, error)
}

func GetName(namer Namer) string {
	nm, _ := namer.GetName()
	return nm
}

func EachChildren(w interface{ GetChildren() *glib.List }, fn func(i int, v interface{}) bool) {
	var cursor int = -1
	for ptr := w.GetChildren(); ptr != nil; ptr = ptr.Next() {
		cursor++

		if fn(cursor, ptr.Data()) {
			return
		}
	}
}

type DragSortable interface {
	DragSourceSet(gdk.ModifierType, []gtk.TargetEntry, gdk.DragAction)
	DragDestSet(gtk.DestDefaults, []gtk.TargetEntry, gdk.DragAction)
	GetAllocation() *gtk.Allocation
	Connector
}

func BindDragSortable(ds DragSortable, target, id string, fn func(id, target string)) {
	var dragEntries = []gtk.TargetEntry{NewTargetEntry(target)}
	var dragAtom = gdk.GdkAtomIntern(target, true)

	// Drag source so you can drag the button away.
	ds.DragSourceSet(gdk.BUTTON1_MASK, dragEntries, gdk.ACTION_MOVE)

	// Drag destination so you can drag the button here.
	ds.DragDestSet(gtk.DEST_DEFAULT_ALL, dragEntries, gdk.ACTION_MOVE)

	ds.Connect("drag-data-get",
		// TODO change ToggleButton.
		func(ds DragSortable, ctx *gdk.DragContext, data *gtk.SelectionData) {
			// Set the index-in-bytes.
			data.SetData(dragAtom, []byte(id))
		},
	)

	ds.Connect("drag-data-received",
		func(ds DragSortable, ctx *gdk.DragContext, x, y uint, data *gtk.SelectionData) {
			// Receive the incoming row's ID and call MoveSession.
			fn(id, string(data.GetData()))
		},
	)

	ds.Connect("drag-begin",
		func(ds DragSortable, ctx *gdk.DragContext) {
			gtk.DragSetIconName(ctx, "user-available-symbolic", 0, 0)
		},
	)
}

type StyleContexter interface {
	GetStyleContext() (*gtk.StyleContext, error)
}

func AddClass(styleCtx StyleContexter, classes ...string) {
	var style, _ = styleCtx.GetStyleContext()
	for _, class := range classes {
		style.AddClass(class)
	}
}

type Bin interface {
	GetChild() (gtk.IWidget, error)
}

var _ Bin = (*gtk.Bin)(nil)

func BinLeftAlignLabel(bin Bin) {
	widget, _ := bin.GetChild()
	widget.(interface{ SetHAlign(gtk.Align) }).SetHAlign(gtk.ALIGN_START)
}

func NewButtonIcon(icon string) *gtk.Image {
	img, _ := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_BUTTON)
	return img
}

func NewImageIconPx(icon string, sizepx int) *gtk.Image {
	img, _ := gtk.ImageNew()
	SetImageIcon(img, icon, sizepx)
	return img
}

func SetImageIcon(img *gtk.Image, icon string, sizepx int) {
	img.SetProperty("icon-name", icon)
	img.SetProperty("pixel-size", sizepx)
	img.SetSizeRequest(sizepx, sizepx)
}

func PrependMenuItems(menu interface{ Prepend(gtk.IMenuItem) }, items []gtk.IMenuItem) {
	for i := len(items) - 1; i >= 0; i-- {
		menu.Prepend(items[i])
	}
}

func AppendMenuItems(menu interface{ Append(gtk.IMenuItem) }, items []gtk.IMenuItem) {
	for _, item := range items {
		menu.Append(item)
	}
}

func HiddenMenuItem(label string, fn interface{}) *gtk.MenuItem {
	mb, _ := gtk.MenuItemNewWithLabel(label)
	mb.Connect("activate", fn)
	return mb
}

func HiddenDisabledMenuItem(label string, fn interface{}) *gtk.MenuItem {
	mb := HiddenMenuItem(label, fn)
	mb.SetSensitive(false)
	return mb
}

func MenuItem(label string, fn interface{}) *gtk.MenuItem {
	menuitem := HiddenMenuItem(label, fn)
	menuitem.Show()
	return menuitem
}

type Connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func BindMenu(connector Connector, menu *gtk.Menu) {
	connector.Connect("event", func(_ *gtk.ToggleButton, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			menu.PopupAtPointer(ev)
		}
	})
}

func BindDynamicMenu(connector Connector, constr func(menu *gtk.Menu)) {
	connector.Connect("event", func(_ *gtk.ToggleButton, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			menu, _ := gtk.MenuNew()
			constr(menu)
			menu.PopupAtPointer(ev)
		}
	})
}

func NewTargetEntry(target string) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, gtk.TARGET_SAME_APP, 0)
	return *e
}

// func 
