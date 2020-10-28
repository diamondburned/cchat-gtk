package dialog

import (
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

type Dialog = gtk.Dialog

type Modal struct {
	*Dialog
	Cancel *gtk.Button
	Action *gtk.Button
	Header *gtk.HeaderBar
}

var headerCSS = primitives.PrepareCSS(`
	.modal-header {
		padding: 0 5px;
	}
`)

func ShowModal(body gtk.IWidget, title, button string, clicked func(m *Modal)) {
	NewModal(body, title, title, clicked).Show()
}

func NewModal(body gtk.IWidget, title, button string, clicked func(m *Modal)) *Modal {
	cancel, _ := gtk.ButtonNewWithMnemonic("_Cancel")
	cancel.SetHAlign(gtk.ALIGN_START)
	cancel.SetRelief(gtk.RELIEF_NONE)
	cancel.Show()

	action, _ := gtk.ButtonNewWithMnemonic(button)
	action.SetHAlign(gtk.ALIGN_END)
	action.SetRelief(gtk.RELIEF_NONE)
	action.Show()

	header, _ := gtk.HeaderBarNew()
	header.SetTitle(title)
	header.PackStart(cancel)
	header.PackEnd(action)
	header.Show()

	primitives.AddClass(header, "modal-header")
	primitives.AttachCSS(header, headerCSS)

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
	dialog.Connect("response", func(_ *gtk.Dialog, resp gtk.ResponseType) {
		if resp == gtk.RESPONSE_DELETE_EVENT {
			dialog.Destroy()
		}
	})
	return dialog
}

func newCSD(body, header gtk.IWidget) *gtk.Dialog {
	dialog, err := gts.NewEmptyModalDialog()
	if err != nil {
		panic(err)
	}
	dialog.SetDefaultSize(450, 300)
	dialog.Add(body)

	if oldh, _ := dialog.GetHeaderBar(); oldh != nil {
		dialog.Remove(oldh)
	}
	dialog.SetTitlebar(header)

	return dialog
}
