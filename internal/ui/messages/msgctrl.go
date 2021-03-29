package messages

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type bindableButton struct {
	gtk.Button
	h glib.SignalHandle
}

func newBindableButton(iconName string) *bindableButton {
	btn, _ := gtk.ButtonNewFromIconName(iconName, iconSize)
	return &bindableButton{
		Button: *btn,
	}
}

func (btn *bindableButton) bind(fn func()) {
	btn.unbind()
	if fn != nil {
		btn.h = btn.Connect("clicked", func(*gtk.Button) { fn() })
		btn.SetSensitive(true)
		btn.Show()
	}
}

func (btn *bindableButton) unbind() {
	if btn.h > 0 {
		btn.HandlerDisconnect(btn.h)
		btn.h = 0
		btn.SetSensitive(false)
		btn.Hide()
	}
}

// MessageItemNames contains names that MessageControl will use for its menu
// action callbacks.
type MessageItemNames struct {
	Reply, Edit, Delete string
}

// MessageControl controls buttons that control a selected message.
type MessageControl struct {
	gtk.Revealer
	Box *gtk.Box

	hide bool

	Reply  *bindableButton
	Edit   *bindableButton
	Delete *bindableButton // Actions "Delete"
}

func NewMessageControl() *MessageControl {
	mc := MessageControl{}

	mc.Reply = newBindableButton("mail-reply-sender-symbolic")
	mc.Edit = newBindableButton("document-edit-symbolic")
	mc.Delete = newBindableButton("edit-delete-symbolic")

	mc.Box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 2)
	mc.Box.Add(mc.Reply)
	mc.Box.Add(mc.Edit)
	mc.Box.Add(mc.Delete)
	mc.Box.Show()

	r, _ := gtk.RevealerNew()
	mc.Revealer = *r
	mc.Revealer.SetTransitionDuration(75)
	mc.Revealer.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_CROSSFADE)
	mc.Revealer.Add(mc.Box)

	mc.Disable()

	return &mc
}

// Enable enables the MessageControl with the given message.
func (mc *MessageControl) Enable(msg container.MessageRow, names MessageItemNames) {
	mc.SetSensitive(true)
	mc.SetRevealChild(true && !mc.hide)

	unwrap := msg.Unwrap(false)

	mc.Reply.bind(menu.FindItemFunc(unwrap.MenuItems, names.Reply))
	mc.Edit.bind(menu.FindItemFunc(unwrap.MenuItems, names.Edit))
	mc.Delete.bind(menu.FindItemFunc(unwrap.MenuItems, names.Delete))
}

// SetHidden sets whether or not the control should be hidden.
func (mc *MessageControl) SetHidden(hidden bool) {
	mc.hide = hidden
}

// Disable disables the MessageControl and hides it.
func (mc *MessageControl) Disable() {
	mc.SetSensitive(false)
	mc.SetRevealChild(false)

	mc.Reply.unbind()
	mc.Edit.unbind()
	mc.Delete.unbind()
}
