package service

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/singlestack"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
)

/*

Design:

____________________________
|  #  |           |        |
|-----|-----------|--------|
|  D  | nixhub    |        |
| --- |   #home   |        | <- shaded revealer
|  O  |   #dev... |        | <- user accounts collapsed
| --- | astolf... |        |
|     | asdada... |        |
|  M  |           |        |
|_____|___________|________|
*/

type Controller interface {
	// SessionSelected is called when
	SessionSelected(svc *Service, srow *session.Row)
	// RowSelected is wrapped around session's MessageRowSelected.
	RowSelected(*session.Row, *server.ServerRow, cchat.ServerMessage)
	// AuthenticateSession is called to spawn the authentication dialog.
	AuthenticateSession(*List, *Service)
	// OnSessionRemove is called to remove a session. This should also clear out
	// the message view in the parent package.
	OnSessionRemove(*Service, *session.Row)
	// OnSessionDisconnect is here to satisfy session's controller.
	OnSessionDisconnect(*Service, *session.Row)
}

type View struct {
	*gtk.Box   // 2 panes, but left-most hard-coded
	Controller // inherit main controller

	Services   *List
	ServerView *gtk.ScrolledWindow

	ServerStack *singlestack.Stack

	// Servers *session.Servers // nil by default; use .Servers
}

func NewView(ctrller Controller) *View {
	view := &View{Controller: ctrller}

	view.Services = NewList(view)
	view.Services.Show()

	// Make a separator.
	// sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	// sep.Show()

	// Make a stack for the middle panel.
	view.ServerStack = singlestack.NewStack()
	view.ServerStack.SetSizeRequest(150, -1) // min width
	view.ServerStack.SetTransitionDuration(50)
	view.ServerStack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	view.ServerStack.SetHomogeneous(true)
	view.ServerStack.Show()

	view.ServerView, _ = gtk.ScrolledWindowNew(nil, nil)
	view.ServerView.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	view.ServerView.Add(view.ServerStack)
	view.ServerView.Show()

	view.Box, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	view.Box.PackStart(view.Services, false, false, 0)
	// view.Box.PackStart(sep, false, false, 0)
	view.Box.PackStart(view.ServerView, true, true, 0)
	view.Box.Show()

	return view
}

func (v *View) AddService(svc cchat.Service) {
	v.Services.AddService(svc)
}

// SessionSelected calls the right-side server view to change.
//
// TODO: think of how to change. Maybe use a stack? Maybe use a box that we
// remove and re-add? does animation matter?
func (v *View) SessionSelected(svc *Service, srow *session.Row) {
	// Unselect every service boxes except this one.
	for _, service := range v.Services.Services {
		if service != svc {
			service.BodyList.UnselectAll()
		}
	}

	// !!!: SHITTY HACK!!!
	// We can do this, as we're keeping all the server lists in memory by Go's
	// reference anyway. In fact, cchat REQUIRES us to do so.
	v.ServerStack.SetVisibleChild(srow.Servers)

	// Call the controller's method.
	v.Controller.SessionSelected(svc, srow)
}
