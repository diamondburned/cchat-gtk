package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/auth"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/gotk3/gotk3/gtk"
)

type View struct {
	*gtk.ScrolledWindow
	Box      *gtk.Box
	Services []*Container
}

func NewView() *View {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Show()
	sw.Add(box)

	return &View{
		sw,
		box,
		nil,
	}
}

func (v *View) AddService(svc cchat.Service) {
	s := NewContainer(svc)
	v.Services = append(v.Services, s)
	v.Box.Add(s)
}

type Container struct {
	*gtk.Box
	header   *header
	revealer *gtk.Revealer
	children *children
}

func NewContainer(svc cchat.Service) *Container {
	header := newHeader(svc)

	children := newChildren()
	chrev, _ := gtk.RevealerNew()
	chrev.Show()
	chrev.SetRevealChild(false)
	chrev.Add(children)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	box.PackStart(header, false, false, 0)
	box.PackStart(chrev, false, false, 0)

	var container = &Container{box, header, chrev, children}

	header.add.Connect("clicked", func() {
		auth.NewDialog(svc.Name(), svc.Authenticate(), container.addSession)
	})

	return container
}

func (c *Container) addSession(ses cchat.Session) {
	srow := session.New(ses)
	c.children.addSessionRow(srow)
}

type header struct {
	*gtk.Box
	label *gtk.Label
	add   *gtk.Button
}

func newHeader(svc cchat.Service) *header {
	label, _ := gtk.LabelNew(svc.Name())
	label.Show()
	label.SetXAlign(0)

	add, _ := gtk.ButtonNewFromIconName("list-add-symbolic", gtk.ICON_SIZE_BUTTON)
	add.SetRelief(gtk.RELIEF_NONE)
	add.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()
	box.PackStart(label, true, true, 5)
	box.PackStart(add, false, false, 0)

	return &header{box, label, add}
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
