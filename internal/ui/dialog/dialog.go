package dialog

import (
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func NewModal(body gtk.IWidget, title, label string, callback func()) *gtk.Dialog {
	cancel, _ := gtk.ButtonNew()
	cancel.Show()
	cancel.SetHAlign(gtk.ALIGN_START)
	cancel.SetRelief(gtk.RELIEF_NONE)
	cancel.SetLabel("Cancel")

	action, _ := gtk.ButtonNew()
	action.Show()
	action.SetHAlign(gtk.ALIGN_END)
	action.SetRelief(gtk.RELIEF_NONE)
	action.SetLabel(label)

	header, _ := gtk.HeaderBarNew()
	header.Show()
	header.SetTitle(title)
	header.PackStart(cancel)
	header.PackEnd(action)

	dialog := newCSD(body, header)

	cancel.Connect("clicked", dialog.Destroy)
	action.Connect("clicked", callback)

	return dialog
}

func NewCSD(body, header gtk.IWidget) *gtk.Dialog {
	dialog := newCSD(body, header)
	dialog.Connect("response", func(_ *glib.Object, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			dialog.Destroy()
		}
	})
	return dialog
}

func newCSD(body, header gtk.IWidget) *gtk.Dialog {
	dialog, _ := gtk.DialogNew()
	dialog.SetModal(true)
	dialog.SetTransientFor(gts.App.Window)

	if area, _ := dialog.GetContentArea(); area != nil {
		dialog.Remove(area)
	}

	if oldh, _ := dialog.GetHeaderBar(); oldh != nil {
		dialog.Remove(oldh)
	}

	dialog.Add(body)

	dialog.SetTitlebar(header)

	return dialog
}
