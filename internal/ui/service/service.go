package service

import (
	"context"
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/drag"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/gotk3/gotk3/gtk"
)

const IconSize = 42

type ListController interface {
	// ClearMessenger is called when a nil slice of servers is set.
	ClearMessenger(*session.Row)
	// MessengerSelected is called when a server message row is clicked.
	MessengerSelected(*session.Row, *server.ServerRow)
	// SessionSelected tells the view to change the session view.
	SessionSelected(*Service, *session.Row)
	// AuthenticateSession tells View to call to the parent's authenticator.
	AuthenticateSession(*Service)
	// MoveService tells the view to shift the service to before the target.
	MoveService(id, targetID string)

	OnSessionRemove(*Service, *session.Row)
	OnSessionDisconnect(*Service, *session.Row)
}

// Service holds everything that a single service has.
type Service struct {
	ListController

	*gtk.Box
	Button *gtk.ToggleButton
	Icon   *roundimage.StillImage
	Menu   *actions.Menu

	BodyRev  *gtk.Revealer // revealed
	BodyList *session.List // not really supposed to be here

	service      cchat.Service // state
	Name         rich.NameContainer
	Configurator cchat.Configurator
}

var serviceCSS = primitives.PrepareClassCSS("service", `
	.service {
		box-shadow: 0 0 2px 0 alpha(@theme_bg_color, 0.75);
		margin: 6px 8px;
		margin-bottom: 0;
		border-radius: 14px;
	}

	.service:first-child { margin-top: 8px; }
	.service:last-child  { margin-bottom: 8px; }
`)

var serviceButtonCSS = primitives.PrepareClassCSS("service-button", `
	.service-button {
		padding: 6px;
		margin:  0;
	}

	.service-button:not(:checked) {
		border-radius: 14px;
		transition: linear 80ms border-radius; /* TODO add delay */
	}

	.service-button:checked {
		border-radius: 14px 14px 0 0;
		/* background-color: alpha(@theme_fg_color, 0.2); */
	}
`)

var serviceIconCSS = primitives.PrepareClassCSS("service-icon", ``)

func NewService(svc cchat.Service, svclctrl ListController) *Service {
	service := Service{
		service:        svc,
		ListController: svclctrl,
	}
	service.Name.QueueNamer(context.Background(), svc)

	service.BodyList = session.NewList(&service)
	service.BodyList.Show()

	service.BodyRev, _ = gtk.RevealerNew()
	service.BodyRev.SetRevealChild(false) // TODO persistent state
	service.BodyRev.SetTransitionDuration(100)
	service.BodyRev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_DOWN)
	service.BodyRev.Add(service.BodyList)
	service.BodyRev.Show()

	// TODO: have it so the button changes to the session avatar when collapsed

	service.Button, _ = gtk.ToggleButtonNew()
	service.Button.SetRelief(gtk.RELIEF_NONE)
	service.Button.Show()
	serviceButtonCSS(service.Button)

	service.Icon = roundimage.NewStillImage(service.Button, 0)
	service.Icon.SetSize(IconSize)
	service.Icon.UseInitialsIfNone(service.Name.String)
	service.Icon.Show()
	serviceIconCSS(service.Icon)

	rich.BindRoundImage(service.Icon, &service.Name, true)
	// TODO: hover for name. We use tooltip for now.

	service.Button.Add(service.Icon)
	service.Button.Connect("clicked", func(tb *gtk.ToggleButton) {
		revealed := !service.GetRevealChild()
		service.SetRevealChild(revealed)
		tb.SetActive(revealed)
	})

	// Bind session.* actions into row.
	service.Menu = actions.NewMenu("service")
	// Bind right clicks and show a popover menu on such event.
	service.Menu.BindRightClick(service.Button)

	if configurator := svc.AsConfigurator(); configurator != nil {
		cfg := config.Configurator{
			Service:      svc,
			Configurator: configurator,
		}
		config.Restore(cfg)
		service.Menu.AddAction("Configure", func() { config.Spawn(cfg) })
	}

	// Intermediary box to contain both the icon and the revealer.
	service.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	service.Box.PackStart(service.Button, false, false, 0)
	service.Box.PackStart(service.BodyRev, false, false, 0)
	service.Box.Show()
	serviceCSS(service.Box)

	// Bind a drag and drop on the button instead of the entire box.
	drag.BindDraggable(&service, "network-workgroup", svclctrl.MoveService, service.Button)

	return &service
}

// SetRevealChild sets whether or not the service should reveal all sessions.
func (s *Service) SetRevealChild(reveal bool) {
	s.BodyRev.SetRevealChild(reveal)
}

// GetRevealChild gets whether or not the service is revealing all sessions.
func (s *Service) GetRevealChild() bool {
	return s.BodyRev.GetRevealChild()
}

func (s *Service) SessionSelected(srow *session.Row) {
	s.ListController.SessionSelected(s, srow)
}

func (s *Service) AuthenticateSession() {
	s.ListController.AuthenticateSession(s)
}

func (s *Service) AddLoadingSession(id, name string) *session.Row {
	if srow := s.BodyList.Session(id); srow != nil {
		return srow
	}

	srow := session.NewLoading(s, id, name, s)
	srow.Show()

	s.BodyList.AddSessionRow(id, srow)
	return srow
}

// AddSession adds the given session. It returns nil if the session already
// exists with the given ID.
func (s *Service) AddSession(ses cchat.Session) *session.Row {
	if srow := s.BodyList.Session(ses.ID()); srow != nil {
		return srow
	}

	srow := session.New(s, ses, s)
	srow.Show()

	s.BodyList.AddSessionRow(ses.ID(), srow)
	s.SaveAllSessions()
	return srow
}

func (s *Service) ID() string {
	return s.service.ID()
}

func (s *Service) Service() cchat.Service {
	return s.service
}

func (s *Service) OnSessionDisconnect(row *session.Row) {
	// Unselect if selected.
	if cur := s.BodyList.GetSelectedRow(); cur.GetIndex() == row.GetIndex() {
		s.BodyList.UnselectAll()
	}

	s.ListController.OnSessionDisconnect(s, row)
}

func (s *Service) RemoveSession(row *session.Row) {
	s.ListController.OnSessionRemove(s, row)
	s.BodyList.RemoveSessionRow(row.ID())
	s.SaveAllSessions()
}

func (s *Service) MoveSession(id, movingID string) {
	s.BodyList.MoveSession(id, movingID)
	s.SaveAllSessions()
}

func (s *Service) Breadcrumb() string {
	return s.Name.String()
}

func (s *Service) ParentBreadcrumb() traverse.Breadcrumber {
	return nil
}

func (s *Service) SaveAllSessions() {
	var sessions = s.BodyList.Sessions()
	var keyrings = keyring.NewService(s.service, len(sessions))

	for _, session := range sessions {
		keyrings.Add(session.Session, s.Name.String())
	}

	keyrings.Save()
}

func (s *Service) RestoreSession(row *session.Row, id string) {
	rs := s.service.AsSessionRestorer()
	if rs == nil {
		return
	}

	if k := keyring.RestoreSession(s.service, id); k != nil {
		row.RestoreSession(rs, *k)
		return
	}

	log.Error(fmt.Errorf(
		"Missing keyring for service %s, session ID %s",
		s.service.ID(), id,
	))
}

// restoreAll restores all sessions.
func (s *Service) restoreAll() {
	rs := s.service.AsSessionRestorer()
	if rs == nil {
		return
	}

	service := keyring.Restore(s.service)

	for _, ses := range service.Sessions {
		// Session is not a pointer, so we can pass it into arguments safely.
		row := s.AddLoadingSession(ses.ID, ses.Name)
		row.RestoreSession(rs, ses)
	}
}
