package dialog

import (
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type Modal struct {
	*gtk.Dialog
	Cancel *gtk.Button
	Action *gtk.Button
	Header *gtk.HeaderBar
}

func ShowModal(body gtk.IWidget, title, button string, clicked func(m *Modal)) {
	NewModal(body, title, title, clicked).Show()
}

func NewModal(body gtk.IWidget, title, button string, clicked func(m *Modal)) *Modal {
	cancel, _ := gtk.ButtonNewWithMnemonic("_Cancel")
	cancel.Show()
	cancel.SetHAlign(gtk.ALIGN_START)
	cancel.SetRelief(gtk.RELIEF_NONE)

	action, _ := gtk.ButtonNewWithMnemonic(button)
	action.Show()
	action.SetHAlign(gtk.ALIGN_END)
	action.SetRelief(gtk.RELIEF_NONE)

	header, _ := gtk.HeaderBarNew()
	header.Show()
	header.SetMarginStart(5)
	header.SetMarginEnd(5)
	header.SetTitle(title)
	header.PackStart(cancel)
	header.PackEnd(action)

	dialog := newCSD(body, header)
	modald := &Modal{
		dialog,
		cancel,
		action,
		header,
	}

	cancel.Connect("clicked", dialog.Destroy)
	action.Connect("clicked", func() { clicked(modald) })

	return modald
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
	dialog, _ := gts.NewEmptyModalDialog()
	dialog.SetDefaultSize(450, 300)
	dialog.Add(body)

	if oldh, _ := dialog.GetHeaderBar(); oldh != nil {
		dialog.Remove(oldh)
	}
	dialog.SetTitlebar(header)

	return dialog
}
