package session

import (
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/spinner"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const FaceSize = 48 // gtk.ICON_SIZE_DIALOG

// Servers wraps around a list of servers inherited from Children. It's the
// container that's displayed on the right of the service sidebar.
type Servers struct {
	*gtk.Box
	Children *server.Children
	spinner  *spinner.Boxed // non-nil if loading.

	// state
	ServerList cchat.ServerList
}

var toplevelCSS = primitives.PrepareClassCSS("top-level", `
	.top-level { margin: 0 3px }
`)

func NewServers(p traverse.Breadcrumber, ctrl server.Controller) *Servers {
	c := server.NewChildren(p, ctrl)
	c.SetMarginStart(0) // children is top level; there is no main row
	c.SetHExpand(true)  // fill
	c.Show()
	toplevelCSS(c)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	return &Servers{
		Box:      b,
		Children: c,
	}
}

func (s *Servers) Reset() {
	// Reset isn't necessarily called while loading, so we do a check.
	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner = nil
	}

	// Reset the state.
	s.ServerList = nil
	// Remove all children.
	primitives.RemoveChildren(s)
	// Reset the children container.
	s.Children.Reset()
}

// IsLoading returns true if the servers container is loading.
func (s *Servers) IsLoading() bool {
	return s.spinner != nil
}

// SetList indicates that the server list has been loaded. Unlike
// server.Children, this method will load immediately.
func (s *Servers) SetList(slist cchat.ServerList) {
	primitives.RemoveChildren(s)
	s.ServerList = slist
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
		err := s.ServerList.Servers(s.Children)
		gts.ExecAsync(func() {
			if err != nil {
				s.setFailed(err)
			} else {
				s.setDone()
			}
		})
	}()
}

// setDone changes the view to show the servers.
func (s *Servers) setDone() {
	primitives.RemoveChildren(s)

	// stop the spinner.
	s.spinner.Stop()
	s.spinner = nil

	s.Add(s.Children)
}

// setLoading shows a loading spinner. Use this after the session row is
// connected.
func (s *Servers) setLoading() {
	primitives.RemoveChildren(s)

	s.spinner = spinner.New()
	s.spinner.SetSizeRequest(FaceSize, FaceSize)
	s.spinner.Show()
	s.spinner.Start()

	s.Add(s.spinner)
}

// setFailed shows a sad face with the error. Use this when the session row has
// failed to load.
func (s *Servers) setFailed(err error) {
	primitives.RemoveChildren(s)

	// stop the spinner. Let this SEGFAULT if nil, as that's undefined behavior.
	s.spinner.Stop()
	s.spinner = nil

	// Create a BLANK label for padding.
	ltop, _ := gtk.LabelNew("")
	ltop.Show()

	// Create a retry button.
	btn, _ := gtk.ButtonNewFromIconName("view-refresh-symbolic", gtk.ICON_SIZE_DIALOG)
	btn.Show()
	btn.Connect("clicked", s.load)

	// Create a bottom label for the error itself.
	lerr, _ := gtk.LabelNew("")
	lerr.SetSingleLineMode(true)
	lerr.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
	lerr.SetMarkup(fmt.Sprintf(
		`<span color="red"><b>Error:</b> %s</span>`,
		humanize.Error(err),
	))
	lerr.Show()

	// Add these items into the box.
	s.PackStart(ltop, false, false, 0)
	s.PackStart(btn, false, false, 10) // pad
	s.PackStart(lerr, false, false, 0)
}
