package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
)

type Controller interface {
	session.Controller

	// MessageRowSelected is wrapped around session's MessageRowSelected.
	MessageRowSelected(*session.Row, *server.Row, cchat.ServerMessage)
	// AuthenticateSession is called to spawn the authentication dialog.
	AuthenticateSession(*Container, cchat.Service)
	// SaveAllSessions is called to save all available sessions from the menu.
	SaveAllSessions(*Container)
}

type View struct {
	*gtk.ScrolledWindow
	Box      *gtk.Box
	Services []*Container
}

func NewView() *View {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	primitives.AddClass(box, "services")

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.Add(box)
	sw.Show()

	return &View{
		sw,
		box,
		nil,
	}
}

func (v *View) AddService(svc cchat.Service, ctrl Controller) *Container {
	s := NewContainer(svc, ctrl)
	v.Services = append(v.Services, s)
	v.Box.Add(s)
	return s
}

type Container struct {
	*gtk.Box
	Service cchat.Service

	header   *header
	revealer *gtk.Revealer
	children *children

	// Embed controller and extend it to override RestoreSession.
	Controller
}

func NewContainer(svc cchat.Service, ctrl Controller) *Container {
	children := newChildren()

	chrev, _ := gtk.RevealerNew()
	chrev.SetRevealChild(true)
	chrev.Add(children)
	chrev.Show()

	header := newHeader(svc)
	header.reveal.SetActive(chrev.GetRevealChild())

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	box.PackStart(header, false, false, 0)
	box.PackStart(chrev, false, false, 0)

	primitives.AddClass(box, "service")

	container := &Container{
		Box:        box,
		Service:    svc,
		header:     header,
		revealer:   chrev,
		children:   children,
		Controller: ctrl,
	}

	// On click, toggle reveal.
	header.reveal.Connect("clicked", func() {
		revealed := !chrev.GetRevealChild()
		chrev.SetRevealChild(revealed)
		header.reveal.SetActive(revealed)
	})

	// On click, show the auth dialog.
	header.add.Connect("clicked", func() {
		ctrl.AuthenticateSession(container, svc)
	})

	// Make menu items.
	primitives.AppendMenuItems(header.Menu, []primitives.MenuItem{
		{Name: "Save Sessions", Fn: func() {
			ctrl.SaveAllSessions(container)
		}},
	})

	return container
}

func (c *Container) AddSession(ses cchat.Session) *session.Row {
	srow := session.New(c, ses, c)
	c.children.addSessionRow(ses.ID(), srow)

	return srow
}

func (c *Container) AddLoadingSession(id, name string) *session.Row {
	srow := session.NewLoading(c, name, c)
	c.children.addSessionRow(id, srow)

	return srow
}

// KeyringSessions returns all known keyring sessions. Sessions that can't be
// saved will not be in the slice.
func (c *Container) KeyringSessions() []keyring.Session {
	var ksessions = make([]keyring.Session, 0, len(c.children.Sessions))
	for _, s := range c.children.Sessions {
		if k := s.KeyringSession(); k != nil {
			ksessions = append(ksessions, *k)
		}
	}
	return ksessions
}

func (c *Container) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(nil, c.header.reveal.GetText())
}
