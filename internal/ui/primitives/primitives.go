package primitives

import (
	"runtime/debug"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type WidgetDestroyer interface {
	gtk.IWidget
	Destroy()
}

type Container interface {
	Remove(gtk.IWidget)
	GetChildren() *glib.List
}

var _ Container = (*gtk.Container)(nil)

// RemoveChildren removes all children from the given container. Most of the
// time, DestroyChildren should be preferred if no children will be reused.
func RemoveChildren(w Container) {
	w.GetChildren().FreeFull(func(child interface{}) {
		w.Remove(child.(gtk.IWidget))
	})
}

// DestroyChildren destroys all children of the given container, removing and
// freeing them at the same time.
func DestroyChildren(w Container) {
	type destroyer interface {
		Destroy()
	}

	w.GetChildren().FreeFull(func(child interface{}) {
		child.(destroyer).Destroy()
	})
}

// ChildrenLen gets the total count of children for the given container.
func ChildrenLen(w Container) int {
	children := w.GetChildren()
	defer children.Free()

	return int(children.Length())
}

func NthChild(w Container, n int) interface{} {
	children := w.GetChildren()
	defer children.Free()

	length := int(children.Length())

	// Bound check!
	if !(0 <= n && n < length) {
		return nil
	}

	return children.NthData(uint(n))
}

// ForeachChildBackwards iterates the list. If the callback returns true, then
// the loop is broken.
func ForeachChild(w Container, fn func(interface{}) (stop bool)) {
	children := w.GetChildren()
	defer children.Free()

	for v := children; v != nil; v = v.Next() {
		if fn(v.Data()) {
			break
		}
	}
}

// ForeachChildBackwards iterates the list backwards. If the callback returns
// true, then the loop is broken.
func ForeachChildBackwards(w Container, fn func(interface{}) (stop bool)) {
	children := w.GetChildren()
	defer children.Free()

	for v := children.Last(); v != nil; v = v.Previous() {
		if fn(v.Data()) {
			break
		}
	}
}

type Namer interface {
	SetName(string)
	GetName() (string, error)
}

func GetName(namer Namer) string {
	nm, _ := namer.GetName()
	return nm
}

func EachChildren(w Container, fn func(i int, v interface{}) bool) {
	var cursor int = -1
	for ptr := w.GetChildren(); ptr != nil; ptr = ptr.Next() {
		cursor++

		if fn(cursor, ptr.Data()) {
			return
		}
	}
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

type ClassEnum struct{ class string }

func (c *ClassEnum) SetClass(ctx StyleContexter, class string) {
	var style, _ = ctx.GetStyleContext()
	if c.class != "" {
		style.RemoveClass(c.class)
	}

	if c.class = class; class != "" {
		style.AddClass(class)
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

type ImageIconSetter interface {
	SetFromIconName(string, gtk.IconSize)
	SetPixelSize(int)
}

func SetImageIcon(img ImageIconSetter, icon string, sizepx int) {
	img.SetFromIconName(icon, gtk.ICON_SIZE_BUTTON)
	img.SetPixelSize(sizepx)
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
	Connect(string, interface{}) glib.SignalHandle
	ConnectAfter(string, interface{}) glib.SignalHandle
	HandlerDisconnect(glib.SignalHandle)
}

var _ Connector = (*glib.Object)(nil)

func OnRightClick(connector Connector, fn func()) {
	connector.Connect("button-press-event", func(c Connector, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			fn()
		}
	})
}

func BindMenu(connector Connector, menu *gtk.Menu) {
	connector.Connect("button-press-event", func(c Connector, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			menu.PopupAtPointer(ev)
		}
	})
}

func BindDynamicMenu(connector Connector, constr func(menu *gtk.Menu)) {
	connector.Connect("button-press-event", func(c Connector, ev *gdk.Event) {
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
		if class != "" {
			s.AddClass(class)
		}
	}
}

func PrepareCSS(css string) *gtk.CssProvider {
	p, _ := gtk.CssProviderNew()
	if err := p.LoadFromData(css); err != nil {
		log.Error(errors.Wrapf(err, "CSS fail at %s", debug.Stack()))
	}
	return p
}

func AttachCSS(ctx StyleContexter, prov *gtk.CssProvider) {
	s, _ := ctx.GetStyleContext()
	s.AddProvider(prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

func InlineCSS(ctx StyleContexter, css string) {
	AttachCSS(ctx, PrepareCSS(css))
}

// LeafletOnFold binds a callback to a leaflet that would be called when the
// leaflet's folded state changes.
func LeafletOnFold(leaflet *handy.Leaflet, foldedFn func(folded bool)) {
	leaflet.ConnectAfter("notify::folded", func(leaflet *handy.Leaflet) {
		foldedFn(leaflet.GetFolded())
	})
}
