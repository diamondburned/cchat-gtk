package spinner

import "github.com/gotk3/gotk3/gtk"

type Boxed struct {
	*gtk.Box
	Spinner *gtk.Spinner
}

func New() *Boxed {
	spin, _ := gtk.SpinnerNew()
	spin.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.SetHAlign(gtk.ALIGN_CENTER)
	box.SetVAlign(gtk.ALIGN_CENTER)
	box.Add(spin)

	return &Boxed{box, spin}
}

func (b *Boxed) SetSizeRequest(w, h int) {
	b.Spinner.SetSizeRequest(w, h)
}

func (b *Boxed) Start() {
	b.Spinner.Start()
}

func (b *Boxed) Stop() {
	b.Spinner.Stop()
}
