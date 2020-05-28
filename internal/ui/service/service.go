package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/auth"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
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

func (v *View) AddService(svc cchat.Service, rowctrl server.RowController) {
	s := NewContainer(svc, rowctrl)
	v.Services = append(v.Services, s)
	v.Box.Add(s)
}

type Container struct {
	*gtk.Box
	header   *header
	revealer *gtk.Revealer
	children *children
	rowctrl  server.RowController
}

func NewContainer(svc cchat.Service, rowctrl server.RowController) *Container {
	header := newHeader(svc)

	children := newChildren()

	chrev, _ := gtk.RevealerNew()
	chrev.SetRevealChild(false)
	chrev.Add(children)
	chrev.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	box.PackStart(header, false, false, 0)
	box.PackStart(chrev, false, false, 0)

	primitives.AddClass(box, "service")

	var container = &Container{box, header, chrev, children, rowctrl}

	// On click, toggle reveal.
	header.reveal.Connect("clicked", func() {
		revealed := !chrev.GetRevealChild()
		chrev.SetRevealChild(revealed)
		header.reveal.SetActive(revealed)
	})

	// On click, show the auth dialog.
	header.add.Connect("clicked", func() {
		auth.NewDialog(svc.Name(), svc.Authenticate(), container.addSession)
	})

	return container
}

func (c *Container) addSession(ses cchat.Session) {
	srow := session.New(ses, c.rowctrl)
	c.children.addSessionRow(srow)
}

type header struct {
	*gtk.Box
	reveal *gtk.ToggleButton
	add    *gtk.Button
}

func newHeader(svc cchat.Service) *header {
	reveal, _ := gtk.ToggleButtonNewWithLabel(svc.Name())
	primitives.BinLeftAlignLabel(reveal) // do this first

	reveal.SetRelief(gtk.RELIEF_NONE)
	reveal.SetMode(true)
	reveal.Show()

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
