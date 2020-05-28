package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Row struct {
	*gtk.Box
	Button  *gtk.ToggleButton
	Session cchat.Session

	Servers *server.Children
}

func New(ses cchat.Session, rowctrl server.RowController) *Row {
	n, err := ses.Name()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to get the username"))
		n = "no name"
	}

	button, _ := gtk.ToggleButtonNewWithLabel(n)
	primitives.BinLeftAlignLabel(button)

	button.SetRelief(gtk.RELIEF_NONE)
	button.Show()

	servers := server.NewChildren(ses, rowctrl)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	box.SetMarginStart(server.ChildrenMargin)
	box.PackStart(button, false, false, 0)
	box.PackStart(servers, false, false, 0)

	primitives.AddClass(box, "session")

	// On click, toggle reveal.
	button.Connect("clicked", func() {
		revealed := !servers.GetRevealChild()
		servers.SetRevealChild(revealed)
		button.SetActive(revealed)
	})

	return &Row{
		Box:     box,
		Button:  button,
		Session: ses,
		Servers: servers,
	}
}
