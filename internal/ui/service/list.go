package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
)

type ViewController interface {
	RowSelected(*session.Row, *server.ServerRow, cchat.ServerMessage)
	SessionSelected(*Service, *session.Row)
	AuthenticateSession(*List, *Service)
}

// List is a list of services. Each service is a revealer that contains
// sessions.
type List struct {
	*gtk.ScrolledWindow

	// same methods as ListController
	ViewController

	ListBox  *gtk.Box
	Services []*Service // TODO: collision check
}

var _ ListController = (*List)(nil)

var listCSS = primitives.PrepareClassCSS("service-list", `
	.service-list {
		padding: 0;
		background-color: mix(@theme_bg_color, @theme_fg_color, 0.03);
	}
`)

func NewList(vctl ViewController) *List {
	svlist := &List{ViewController: vctl}

	// List box of buttons.
	svlist.ListBox, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	svlist.ListBox.Show()
	listCSS(svlist.ListBox)

	svlist.ScrolledWindow, _ = gtk.ScrolledWindowNew(nil, nil)
	svlist.ScrolledWindow.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL)
	svlist.ScrolledWindow.Add(svlist.ListBox)

	return svlist
}

func (sl *List) SetSizeRequest(w, h int) {
	sl.ScrolledWindow.SetSizeRequest(w, h)
	sl.ListBox.SetSizeRequest(w, h)
}

func (sl *List) AuthenticateSession(svc *Service) {
	sl.ViewController.AuthenticateSession(sl, svc)
}

func (sl *List) AddService(svc cchat.Service) {
	row := NewService(svc, sl)
	row.Show()

	sl.ListBox.Add(row)
	sl.Services = append(sl.Services, row)

	// Try and restore all sessions.
	row.restoreAll()

	// TODO: drag-and-drop?
}

/*
type View struct {
	*gtk.ScrolledWindow
	Box      *gtk.Box
	Services []*Container
}

var servicesCSS = primitives.PrepareCSS(`
	.services {
		background-color: @theme_base_color;
	}
`)

func NewView() *View {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()

	primitives.AddClass(box, "services")
	primitives.AttachCSS(box, servicesCSS)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.Add(box)

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
	// RowSelected is wrapped around session's MessageRowSelected.
	RowSelected(*session.Row, *server.ServerRow, cchat.ServerMessage)
	// AuthenticateSession is called to spawn the authentication dialog.
	AuthenticateSession(*Container, cchat.Service)
	// OnSessionRemove is called to remove a session. This should also clear out
	// the message view in the parent package.
	OnSessionRemove(id string)
	// OnSessionDisconnect is here to satisfy session's controller.
	OnSessionDisconnect(id string)
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
	header.SetActive(chrev.GetRevealChild())

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
	header.Connect("clicked", func() {
		revealed := !chrev.GetRevealChild()
		chrev.SetRevealChild(revealed)
		header.SetActive(revealed)
	})

	// On click, show the auth dialog.
	header.Add.Connect("clicked", func() {
		ctrl.AuthenticateSession(container, svc)
	})

	// Add more menu item(s).
	header.Menu.AddSimpleItem("Save Sessions", container.SaveAllSessions)

	return container
}

func (c *Container) GetService() cchat.Service {
	return c.Service
}

func (c *Container) Sessions() []*session.Row {
	return c.children.Sessions()
}

func (c *Container) AddSession(ses cchat.Session) *session.Row {
	srow := session.New(c, ses, c)
	c.children.AddSessionRow(ses.ID(), srow)
	c.SaveAllSessions()
	return srow
}

func (c *Container) AddLoadingSession(id, name string) *session.Row {
	srow := session.NewLoading(c, id, name, c)
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

func (c *Container) OnSessionDisconnect(ses *session.Row) {
	c.Controller.OnSessionDisconnect(ses.ID())
}

// RestoreSession tries to restore sessions asynchronously. This satisfies
// session.Controller.
func (c *Container) RestoreSession(row *session.Row, id string) {
	// Can this session be restored? If not, exit.
	restorer, ok := c.Service.(cchat.SessionRestorer)
	if !ok {
		return
	}

	// Do we even have a session stored?
	krs := keyring.RestoreSession(c.Service.Name(), id)
	if krs == nil {
		log.Error(fmt.Errorf(
			"Missing keyring for service %s, session ID %s",
			c.Service.Name().Content, id,
		))

		return
	}

	c.restoreSession(row, restorer, *krs)
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

			gts.ExecAsync(func() { r.SetFailed(err) })
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
	return breadcrumb.Try(nil, c.header.GetText())
}
*/
