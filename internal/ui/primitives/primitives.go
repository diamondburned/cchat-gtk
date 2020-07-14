package primitives

import (
	"path/filepath"
	"runtime"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Container interface {
	Remove(gtk.IWidget)
	GetChildren() *glib.List
}

var _ Container = (*gtk.Container)(nil)

func RemoveChildren(w Container) {
	w.GetChildren().Foreach(func(child interface{}) {
		w.Remove(child.(gtk.IWidget))
	})
}

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

func RemoveClass(styleCtx StyleContexter, classes ...string) {
	var style, _ = styleCtx.GetStyleContext()
	for _, class := range classes {
		style.RemoveClass(class)
	}
}

type StyleContextFocuser interface {
	StyleContexter
	GrabFocus()
}

// SuggestAction styles the element to have the suggeested action class.
func SuggestAction(styleCtx StyleContextFocuser) {
	AddClass(styleCtx, "suggested-action")
	styleCtx.GrabFocus()
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
	connector.Connect("button-press-event", func(_ *gtk.ToggleButton, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			menu.PopupAtPointer(ev)
		}
	})
}

func BindDynamicMenu(connector Connector, constr func(menu *gtk.Menu)) {
	connector.Connect("button-press-event", func(_ *gtk.ToggleButton, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			menu, _ := gtk.MenuNew()
			constr(menu)

			// Only show the menu if the callback added any children into the
			// list.
			if menu.GetChildren().Length() > 0 {
				menu.PopupAtPointer(ev)
			}
		}
	})
}

func NewTargetEntry(target string) gtk.TargetEntry {
	e, _ := gtk.TargetEntryNew(target, gtk.TARGET_SAME_APP, 0)
	return *e
}

// NewMenuActionButton is the same as NewActionButton, but it uses the
// open-menu-symbolic icon.
func NewMenuActionButton(actions [][2]string) *gtk.MenuButton {
	return NewActionButton("open-menu-symbolic", actions)
}

// NewActionButton creates a new menu button that spawns a popover with the
// listed actions.
func NewActionButton(iconName string, actions [][2]string) *gtk.MenuButton {
	p, _ := gtk.PopoverNew(nil)
	p.SetSizeRequest(200, -1) // wide enough width
	ActionPopover(p, actions)

	i, _ := gtk.ImageNew()
	i.SetProperty("icon-name", iconName)
	i.SetProperty("icon-size", gtk.ICON_SIZE_SMALL_TOOLBAR)
	i.Show()

	b, _ := gtk.MenuButtonNew()
	b.SetHAlign(gtk.ALIGN_CENTER)
	b.SetPopover(p)
	b.Add(i)

	return b
}

// LabelTweaker is used for ActionPopover and other functions that may need to
// change the alignment of children widgets.
type LabelTweaker interface {
	SetUseMarkup(bool)
	SetHAlign(gtk.Align)
	SetXAlign(float64)
}

var _ LabelTweaker = (*gtk.Label)(nil)

func ActionPopover(p *gtk.Popover, actions [][2]string) {
	var box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)

	for _, action := range actions {
		b, _ := gtk.ModelButtonNew()
		b.SetLabel(action[0])
		b.SetActionName(action[1])
		b.Show()

		// Set the label's alignment in a hacky way.
		c, _ := b.GetChild()
		l := c.(LabelTweaker)
		l.SetUseMarkup(true)
		l.SetHAlign(gtk.ALIGN_START)

		box.PackStart(b, false, true, 0)
	}

	box.Show()
	p.Add(box)
}

func PrepareClassCSS(class, css string) (attach func(StyleContexter)) {
	prov := PrepareCSS(css)

	return func(ctx StyleContexter) {
		s, _ := ctx.GetStyleContext()
		s.AddProvider(prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
		s.AddClass(class)
	}
}

func PrepareCSS(css string) *gtk.CssProvider {
	p, _ := gtk.CssProviderNew()
	if err := p.LoadFromData(css); err != nil {
		_, fn, caller, _ := runtime.Caller(1)
		fn = filepath.Base(fn)
		log.Error(errors.Wrapf(err, "CSS fail at %s:%d", fn, caller))
	}
	return p
}

func AttachCSS(ctx StyleContexter, prov *gtk.CssProvider) {
	s, _ := ctx.GetStyleContext()
	s.AddProvider(prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
