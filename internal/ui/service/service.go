package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

const IconSize = 32

type Controller interface {
	session.Controller
	AuthenticateSession(*Container, cchat.Service)
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
	header   *header
	revealer *gtk.Revealer
	children *children
	rowctrl  Controller
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

	var container = &Container{box, header, chrev, children, ctrl}

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

	return container
}

func (c *Container) AddSession(ses cchat.Session) {
	srow := session.New(ses, c.rowctrl)
	c.children.addSessionRow(srow)
}

func (c *Container) Sessions() []cchat.Session {
	var sessions = make([]cchat.Session, len(c.children.Sessions))
	for i, s := range c.children.Sessions {
		sessions[i] = s.Session
	}
	return sessions
}

type header struct {
	*gtk.Box
	reveal *rich.ToggleButtonImage // no rich text here but it's left aligned
	add    *gtk.Button
}

func newHeader(svc cchat.Service) *header {
	reveal := rich.NewToggleButtonImage(text.Rich{Content: svc.Name()}, "")
	reveal.Box.SetHAlign(gtk.ALIGN_START)
	reveal.SetRelief(gtk.RELIEF_NONE)
	reveal.SetMode(true)
	reveal.Show()

	// Set a custom icon.
	primitives.SetImageIcon(&reveal.Image, "folder-remote-symbolic", IconSize)

	add, _ := gtk.ButtonNewFromIconName("list-add-symbolic", gtk.ICON_SIZE_BUTTON)
	add.SetRelief(gtk.RELIEF_NONE)
	add.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(reveal, true, true, 0)
	box.PackStart(add, false, false, 0)
	box.Show()

	return &header{box, reveal, add}
}

type children struct {
	*gtk.Box
	Sessions []*session.Row
}

func newChildren() *children {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	return &children{box, nil}
}

func (c *children) addSessionRow(row *session.Row) {
	c.Sessions = append(c.Sessions, row)
	c.Box.Add(row)
}
