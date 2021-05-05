package session

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/spinner"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/serverpane"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

const FaceSize = 48 // gtk.ICON_SIZE_DIALOG
const ListWidth = 200

// SessionController extends server.Controller to add needed methods that the
// specific top-level servers container needs.
type SessionController interface {
	ClearMessenger()
	MessengerSelected(*server.ServerRow)
}

// Servers wraps around a list of servers inherited from Children to display a
// Lister in its own box instead of as a nested list. It's the container that's
// displayed on the right of the service sidebar.
type Servers struct {
	gtk.Stack
	SessionController

	spinner *spinner.Boxed
	// Main is the horizontal box containing the current struct's list of
	// servers columnated with the same level. The second item in the box should
	// be the selected server.
	Main *serverpane.Paned

	// Lister is the current lister belonging to this server.
	Lister cchat.Lister
	stopLs func()

	// Children is main's lhs.
	Children *server.Children

	// NextColumn is main's rhs.
	NextColumn *Servers // nil
	detachNext func()
}

var toplevelCSS = primitives.PrepareClassCSS("top-level", `
	.top-level {
	}
`)

// NewServers creates a new Servers instance that holds only the given column
// number and its children. Any servers with a different columnate ID will be in
// the children pane.
func NewServers(p traverse.Breadcrumber, ctrl SessionController) *Servers {
	servers := Servers{
		SessionController: ctrl,
	}

	servers.Children = server.NewChildren(p, &servers)
	servers.Children.SetVExpand(true)
	servers.Children.Show()
	toplevelCSS(servers.Children)

	servers.Main = serverpane.NewPaned(servers.Children, gtk.ORIENTATION_VERTICAL)
	servers.Main.Show()

	stack, _ := gtk.StackNew()
	servers.Stack = *stack
	servers.Stack.SetVAlign(gtk.ALIGN_START)
	servers.Stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	servers.Stack.SetTransitionDuration(75)
	servers.Stack.AddNamed(servers.Main, "main")
	servers.Stack.Show()

	return &servers
}

// Destroy destroys and invalidates this instance permanently.
func (s *Servers) Destroy() {
	s.Reset()
	s.Stack.Destroy()
}

func (s *Servers) Reset() {
	// Reset isn't necessarily called while loading, so we do a check.
	if s.spinner != nil {
		s.spinner.Destroy()
		s.spinner = nil
	}

	// Close the right server column if any.
	if s.NextColumn != nil {
		if s.detachNext != nil {
			s.detachNext()
		}

		s.Main.Remove(s.NextColumn)
		s.NextColumn.Destroy()
		s.NextColumn = nil
	}

	// Call the destructor if any.
	if s.stopLs != nil {
		s.stopLs()
		s.stopLs = nil
	}

	// Reset the state.
	s.Lister = nil
	// Reset the children container.
	s.Children.Reset()
	s.Stack.SetVisibleChild(s.Main)
}

// IsLoading returns true if the servers container is loading.
func (s *Servers) IsLoading() bool {
	return s.spinner != nil
}

// SetList indicates that the server list has been loaded. Unlike
// server.Children, this method will load immediately.
func (s *Servers) SetList(slist cchat.Lister) {
	if s.stopLs != nil {
		s.stopLs()
		s.stopLs = nil
	}

	s.Lister = slist
	s.load()
}

func (s *Servers) load() {
	// Return if we're loading.
	if s.IsLoading() {
		return
	}

	// Mark the servers list as loading.
	s.setLoading()

	go func() {
		stop, err := s.Lister.Servers(s)
		gts.ExecAsync(func() {
			if err != nil {
				s.setFailed(err)
			} else {
				s.stopLs = stop
				s.setDone()
			}
		})
	}()
}

// SetServers is reserved for cchat.ServersContainer.
func (s *Servers) SetServers(servers []cchat.Server) {
	gts.ExecAsync(func() {
		s.Children.SetServersUnsafe(servers)

		if servers == nil {
			s.ClearMessenger()
			return
		}

		// Reload all top-level nodes.
		s.Children.LoadAll()
	})
}

// SetServers is reserved for cchat.ServersContainer.
func (s *Servers) UpdateServer(update cchat.ServerUpdate) {
	gts.ExecAsync(func() { s.Children.UpdateServerUnsafe(update) })
}

// setDone changes the view to show the servers.
func (s *Servers) setDone() {
	s.SetVisibleChild(s.Main)

	// stop the spinner.
	if s.spinner != nil {
		s.spinner.Destroy()
		s.spinner = nil
	}
}

// setLoading shows a loading spinner. Use this after the session row is
// connected.
func (s *Servers) setLoading() {
	s.spinner = spinner.New()
	s.spinner.SetSizeRequest(FaceSize, FaceSize)
	s.spinner.Show()
	s.spinner.Start()

	s.AddNamed(s.spinner, "spinner")
	s.SetVisibleChildName("spinner")
}

// setFailed shows a sad face with the error. Use this when the session row has
// failed to load.
func (s *Servers) setFailed(err error) {
	// stop the spinner. Let this SEGFAULT if nil.
	s.spinner.Destroy()
	s.spinner = nil

	// Remove existing error widgets.
	w, err := s.Stack.GetChildByName("error")
	if err == nil {
		s.Stack.Remove(w)
	}

	// Create a retry button.
	btn, _ := gtk.ButtonNewFromIconName("view-refresh-symbolic", gtk.ICON_SIZE_BUTTON)
	btn.SetLabel("Retry")
	btn.Connect("clicked", s.load)
	btn.Show()

	page := handy.StatusPageNew()
	page.SetTitle("Error")
	page.SetIconName("dialog-error")
	page.SetTooltipText(err.Error())
	page.SetDescription(humanize.Error(err))
	page.Add(btn)

	s.Stack.AddNamed(page, "error")
	s.Stack.SetVisibleChildName("error")
}

// SelectColumnatedLister is called by children servers to open up a server list
// on the right.
func (s *Servers) SelectColumnatedLister(srv *server.ServerRow, lst cchat.Lister) {
	if s.detachNext != nil {
		s.Main.Remove(s.NextColumn) // run the deconstructor
		s.detachNext()
		s.NextColumn.Destroy()
	}

	if lst == nil {
		return
	}

	s.NextColumn = NewServers(srv, s)
	s.NextColumn.SetList(lst)
	s.Main.AddSide(s.NextColumn)

	update := func(box *gtk.Box) {
		a := box.GetAllocation()
		// Align the next column to the selected item.
		s.NextColumn.SetMarginTop(a.GetY())
	}

	s.Children.SetExpand(false)
	primitives.AddClass(srv, "active-column")

	update(srv.Box)
	sizeHandle := srv.Box.Connect("size-allocate", update)

	s.detachNext = func() {
		srv.Box.HandlerDisconnect(sizeHandle)

		s.Children.SetExpand(true)
		primitives.RemoveClass(srv, "active-column")
	}
}
