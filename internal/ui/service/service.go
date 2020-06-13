package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
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

func (v *View) AddService(svc cchat.Service, ctrl Controller) *Container {
	s := NewContainer(svc, ctrl)
	v.Services = append(v.Services, s)
	v.Box.Add(s)

	// Try and restore all sessions.
	s.restoreAllSessions()

	return s
}

type Controller interface {
	// MessageRowSelected is wrapped around session's MessageRowSelected.
	MessageRowSelected(*session.Row, *server.Row, cchat.ServerMessage)
	// AuthenticateSession is called to spawn the authentication dialog.
	AuthenticateSession(*Container, cchat.Service)
	// OnSessionRemove is called to remove a session. This should also clear out
	// the message view in the parent package.
	OnSessionRemove(id string)
}

// Container represents a single service, including the button header and the
// child containers.
type Container struct {
	*gtk.Box
	Service cchat.Service

	header   *header
	revealer *gtk.Revealer
	children *children

	// Embed controller and extend it to override RestoreSession.
	Controller
}

// Guarantee that our interface is up-to-date with session's controller.
var _ session.Controller = (*Container)(nil)

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
	primitives.AppendMenuItems(header.Menu, []*gtk.MenuItem{
		primitives.MenuItem("Save Sessions", func() {
			container.SaveAllSessions()
		}),
	})

	return container
}

func (c *Container) AddSession(ses cchat.Session) *session.Row {
	srow := session.New(c, ses, c)
	c.children.AddSessionRow(ses.ID(), srow)
	c.SaveAllSessions()
	return srow
}

func (c *Container) AddLoadingSession(id, name string) *session.Row {
	srow := session.NewLoading(c, name, c)
	c.children.AddSessionRow(id, srow)
	return srow
}

func (c *Container) RemoveSession(row *session.Row) {
	var id = row.Session.ID()
	c.children.RemoveSessionRow(id)
	c.SaveAllSessions()
	// Call the parent's method.
	c.Controller.OnSessionRemove(id)
}

func (c *Container) MoveSession(rowID, beneathRowID string) {
	c.children.MoveSession(rowID, beneathRowID)
	c.SaveAllSessions()
}

// RestoreSession tries to restore sessions asynchronously. This satisfies
// session.Controller.
func (c *Container) RestoreSession(row *session.Row, krs keyring.Session) {
	// Can this session be restored? If not, exit.
	restorer, ok := c.Service.(cchat.SessionRestorer)
	if !ok {
		return
	}
	c.restoreSession(row, restorer, krs)
}

// internal method called on AddService.
func (c *Container) restoreAllSessions() {
	// Can this session be restored? If not, exit.
	restorer, ok := c.Service.(cchat.SessionRestorer)
	if !ok {
		return
	}

	var sessions = keyring.RestoreSessions(c.Service.Name())

	for _, krs := range sessions {
		// Copy the session to avoid race conditions.
		krs := krs
		row := c.AddLoadingSession(krs.ID, krs.Name)

		c.restoreSession(row, restorer, krs)
	}
}

func (c *Container) restoreSession(r *session.Row, res cchat.SessionRestorer, k keyring.Session) {
	go func() {
		s, err := res.RestoreSession(k.Data)
		if err != nil {
			err = errors.Wrapf(err, "Failed to restore session %s (%s)", k.ID, k.Name)
			log.Error(err)

			gts.ExecAsync(func() { r.SetFailed(k, err) })
		} else {
			gts.ExecAsync(func() { r.SetSession(s) })
		}
	}()
}

func (c *Container) SaveAllSessions() {
	var sessions = c.children.Sessions()
	var ksessions = make([]keyring.Session, 0, len(sessions))

	for _, s := range sessions {
		if k := s.KeyringSession(); k != nil {
			ksessions = append(ksessions, *k)
		}
	}

	keyring.SaveSessions(c.Service.Name(), ksessions)
}

func (c *Container) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(nil, c.header.reveal.GetText())
}
