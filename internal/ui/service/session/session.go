package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type Row struct {
	*gtk.Box
	Button  *gtk.Button
	Session cchat.Session

	Servers *server.Children
}

func New(ses cchat.Session) *Row {
	n, err := ses.Name()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to get the username"))
		n = "no name"
	}

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()

	button, _ := gtk.ButtonNew()
	button.Show()
	button.SetRelief(gtk.RELIEF_NONE)
	button.SetLabel(n)

	rev, _ := gtk.RevealerNew()
	rev.Show()
	rev.SetRevealChild(false)

	return &Row{
		Box:     box,
		Button:  button,
		Session: ses,
		Servers: server.NewChildren(ses),
	}
}
