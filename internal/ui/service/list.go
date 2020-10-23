package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/drag"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
)

type ViewController interface {
	MessengerSelected(*session.Row, *server.ServerRow)
	SessionSelected(*Service, *session.Row)
	AuthenticateSession(*List, *Service)
	OnSessionRemove(*Service, *session.Row)
	OnSessionDisconnect(*Service, *session.Row)
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
	svlist.ListBox.SetHAlign(gtk.ALIGN_START)
	listCSS(svlist.ListBox)

	svlist.ScrolledWindow, _ = gtk.ScrolledWindowNew(nil, nil)
	svlist.ScrolledWindow.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_EXTERNAL)
	svlist.ScrolledWindow.SetHExpand(false)
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

func (sl *List) OnSessionRemove(svc *Service, row *session.Row) {
	sl.ViewController.OnSessionRemove(svc, row)
}

func (sl *List) OnSessionDisconnect(svc *Service, row *session.Row) {
	sl.ViewController.OnSessionDisconnect(svc, row)
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

func (sl *List) MoveService(targetID, movingID string) {
	// Find the widgets.
	var movingsv *Service
	for _, svc := range sl.Services {
		if svc.ID() == movingID {
			movingsv = svc
		}
	}

	// Not found, return.
	if movingsv == nil {
		return
	}

	// Get the location of where to move the widget to.
	var targetix = drag.Find(sl.ListBox, targetID)

	// Actually move the child.
	sl.ListBox.ReorderChild(movingsv, targetix)
}
