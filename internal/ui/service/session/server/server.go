package server

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/savepath"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/button"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/commander"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 0 // refer to style.css

func AssertUnhollow(hollower interface{ IsHollow() bool }) {
	if hollower.IsHollow() {
		panic("Server is hollow, but a normal method was called.")
	}
}

// ParentController controls ServerRow's container, which is the Children
// struct.
type ParentController interface {
	Controller
	ForceIcons()
}

type ServerRow struct {
	*gtk.Box
	Button      *button.ToggleButton
	ActionsMenu *actions.Menu

	Server cchat.Server
	name   rich.NameContainer
	ctrl   ParentController

	parentcrumb traverse.Breadcrumber

	cmder *commander.Buffer

	// non-nil if server list and the function returns error
	childrenErr error

	childrev   *gtk.Revealer
	children   *Children
	serverList cchat.Lister
	serverStop func()

	// State that's updated even when stale. Initializations will use these.
	unread    bool
	mentioned bool
	showLabel bool

	// callback to cancel unread indicator
	cancelUnread func()
}

var serverCSS = primitives.PrepareClassCSS("server", `
	/* Ignore first child because .server-children already covers this */
	.server:not(:first-child) {
		margin: 0;
		border-radius: 0;
	}

	.server.active-column {
		background-color: mix(@theme_bg_color, @theme_selected_bg_color, 0.25);
	}
`)

// NewHollowServer creates a new hollow ServerRow. It will automatically create
// hollow children containers and rows for the given server.
func NewHollowServer(p traverse.Breadcrumber, sv cchat.Server, ctrl ParentController) *ServerRow {
	serverRow := ServerRow{
		parentcrumb:  p,
		ctrl:         ctrl,
		Server:       sv,
		cancelUnread: func() {},
	}

	serverRow.name.QueueNamer(context.Background(), sv)

	lister := sv.AsLister()
	messenger := sv.AsMessenger()

	switch {
	case lister != nil:
		serverRow.SetHollowServerList(lister, ctrl)
		serverRow.children.SetUnreadHandler(serverRow.SetUnreadUnsafe)

	case messenger != nil:
		if unreader := messenger.AsUnreadIndicator(); unreader != nil {
			gts.Async(func() (func(), error) {
				c, err := unreader.UnreadIndicate(&serverRow)
				if err != nil {
					return nil, errors.Wrap(err, "Failed to use unread indicator")
				}

				return func() { serverRow.cancelUnread = c }, nil
			})
		}
	}

	return &serverRow
}

// Init brings the row out of the hollow state. It loads the children (if any),
// but this process does not make more widgets.
func (r *ServerRow) Init() {
	if !r.IsHollow() {
		return
	}

	// Initialize the row, which would fill up the button and others as well.

	r.Button = button.NewToggleButton(&r.name)
	r.Button.SetShowLabel(r.showLabel)
	r.Button.Show()

	r.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	r.Box.SetHAlign(gtk.ALIGN_FILL)
	r.Box.PackStart(r.Button, false, false, 0)
	serverCSS(r.Box)

	// Make the Actions menu.
	r.ActionsMenu = actions.NewMenu("server")
	r.ActionsMenu.InsertActionGroup(r)

	// Ensure errors are displayed.
	r.childrenSetErr(r.childrenErr)

	// Connect the destroyer, if any.
	r.Connect("destroy", func(interface{}) { r.cancelUnread() })

	// Restore the read state.
	r.Button.SetUnreadUnsafe(r.unread, r.mentioned) // update with state

	if cmder := r.Server.AsCommander(); cmder != nil {
		r.cmder = commander.NewBuffer(&r.name, cmder)
		r.ActionsMenu.AddAction("Command Prompt", r.cmder.ShowDialog)
	}

	// Bind right clicks and show a popover menu on such event.
	r.Button.Connect("button-press-event", func(_ gtk.IWidget, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			r.ActionsMenu.Popup(r)
		}
	})

	// Bring up the icons of all the current level's rows if we have one.
	r.name.OnUpdate(func() {
		if r.name.Image().HasImage() {
			r.ctrl.ForceIcons()
		}
	})

	var (
		lister    = r.Server.AsLister()
		columnate = lister != nil && lister.Columnate()
		messenger = r.Server.AsMessenger()
	)

	switch {
	case lister != nil && !columnate:
		primitives.AddClass(r, "server-list")
		r.children.Init()
		r.children.Show()

		r.childrev, _ = gtk.RevealerNew()
		r.childrev.SetRevealChild(false)
		r.childrev.Add(r.children)
		r.childrev.Show()

		r.Box.PackStart(r.childrev, false, false, 0)
		r.Button.SetClicked(r.SetRevealChild)

	case lister != nil && columnate:
		primitives.AddClass(r, "server-list")
		primitives.AddClass(r, "server-columnate")
		r.Button.SetClicked(func(active bool) {
			if active {
				r.ctrl.SelectColumnatedLister(r, lister)
			} else {
				r.ctrl.SelectColumnatedLister(r, nil)
			}
		})

	case messenger != nil:
		primitives.AddClass(r, "server-message")
		r.Button.SetClicked(func(active bool) {
			if active {
				r.ctrl.MessengerSelected(r)
			} else {
				r.ctrl.ClearMessenger()
			}
		})
	}

	// Restore the label visibility state.
	r.SetShowLabel(r.showLabel)
}

// IsActiveServerMessage returns true if the row is currently selected AND it
// is a message row.
func (r *ServerRow) IsActiveServerMessage() bool {
	// If the button is nil, then that probably means we're still in a hollow
	// state. This obviously means nothing is being selected.
	if r.Button == nil {
		return false
	}

	return r.children == nil && r.Button.GetActive()
}

// SetUnread is thread-safe.
func (r *ServerRow) SetUnread(unread, mentioned bool) {
	gts.ExecAsync(func() { r.SetUnreadUnsafe(unread, mentioned) })
}

func (r *ServerRow) SetUnreadUnsafe(unread, mentioned bool) {
	// We're never unread if we're reading this current server.
	if r.IsActiveServerMessage() {
		unread, mentioned = false, false
	}

	// Update the local state.
	r.unread = unread
	r.mentioned = mentioned

	// Button is nil if we're still in a hollow state. A nil check should tell
	// us that.
	if r.Button != nil {
		r.Button.SetUnreadUnsafe(r.unread, r.mentioned)
	}

	// Still update the parent's state even if we're hollow.
	traverse.TrySetUnread(r.parentcrumb, r.Server.ID(), r.unread, r.mentioned)
}

func (r *ServerRow) IsHollow() bool {
	return r.Box == nil
}

// SetHollowServerList sets the row to a hollow server list (children) and
// recursively create
func (r *ServerRow) SetHollowServerList(list cchat.Lister, ctrl Controller) {
	r.serverList = list
	r.children = NewHollowChildren(r, ctrl)
	r.load(func(error) {})
}

// load calls finish if the server list is not loaded. If it is, finish is
// called with a nil immediately.
func (r *ServerRow) load(finish func(error)) {
	if r.children.Rows != nil {
		finish(nil)
		return
	}

	list := r.serverList
	children := r.children
	children.setLoading()

	if !r.IsHollow() {
		r.SetSensitive(false)
	}

	go func() {
		stop, err := list.Servers(children)
		if err != nil {
			log.Error(errors.Wrap(err, "Failed to get servers"))
		}

		gts.ExecAsync(func() {
			r.serverStop = stop

			// Announce that we're not loading anymore.
			r.children.setNotLoading()

			if !r.IsHollow() {
				// Restore clickability.
				r.SetSensitive(true)
			}

			// Use the childrenX method instead of SetX. We can wrap nil
			// errors.
			r.childrenSetErr(errors.Wrap(err, "Failed to get servers"))

			finish(err)
		})
	}()
}

// Reset clears off all children servers. It's a no-op if there are none.
func (r *ServerRow) Reset() {
	if r.children != nil {
		r.children.Reset()
		r.children.Destroy()
	}

	if r.serverStop != nil {
		r.serverStop()
		r.serverStop = nil
	}

	// Reset the state.
	r.ActionsMenu.Reset()
	r.serverList = nil
	r.children = nil
}

func (r *ServerRow) childrenSetErr(err error) {
	// Update the state and only use this state field.
	r.childrenErr = err

	// Only call this if we're not hollow. If we are, then Init() will read the
	// state field above and render the failed button.
	if !r.IsHollow() {
		if err != nil {
			// If the user chooses to retry, the list will automatically expand.
			r.SetFailed(err, func() { r.SetRevealChild(true) })
		} else {
			r.SetDone()
		}
	}
}

// UseEmptyIcon forces the row to show a placeholder icon.
func (r *ServerRow) UseEmptyIcon() {
	AssertUnhollow(r)
	r.Button.UseEmptyIcon()
}

// HasIcon returns true if the current row has an icon.
func (r *ServerRow) HasIcon() bool {
	return !r.IsHollow() && r.Button.Image != nil
}

// SetShowLabel sets whether or not to show the button's (and its children's, if
// any)'s icons.
func (r *ServerRow) SetShowLabel(showLabel bool) {
	r.showLabel = showLabel

	if r.IsHollow() {
		return
	}

	r.Button.SetShowLabel(showLabel)

	// We'd want the button to be wide if we're showing the label. Otherwise,
	// it can be small.
	if r.Button.GetShowLabel() {
		r.Box.SetHAlign(gtk.ALIGN_FILL)
	} else {
		r.Box.SetHAlign(gtk.ALIGN_START)
	}

	if r.children != nil && !r.children.IsHollow() {
		r.children.SetExpand(showLabel)
	}
}

func (r *ServerRow) ParentBreadcrumb() traverse.Breadcrumber {
	return r.parentcrumb
}

func (r *ServerRow) Breadcrumb() string {
	if r.IsHollow() {
		return ""
	}

	return r.name.String()
}

// ID returns the server ID.
func (r *ServerRow) ID() cchat.ID {
	return r.Server.ID()
}

// Name returns the name state.
func (r *ServerRow) Name() rich.LabelStateStorer {
	return &r.name
}

// SetLoading is called by the parent struct.
func (r *ServerRow) SetLoading() {
	AssertUnhollow(r)

	r.SetSensitive(false)
	r.Button.SetLoading()
}

// SetFailed is shared between the parent struct and the children list. This is
// because both of those errors share the same appearance, just different
// callbacks.
func (r *ServerRow) SetFailed(err error, retry func()) {
	AssertUnhollow(r)

	r.SetSensitive(true)
	r.SetTooltipText(err.Error())
	r.Button.SetFailed(err, retry)
	r.ActionsMenu.Reset()
	r.ActionsMenu.AddAction("Retry", retry)
}

// SetDone is shared between the parent struct and the children list. This is
// because both will use the same SetFailed.
func (r *ServerRow) SetDone() {
	AssertUnhollow(r)

	r.Button.SetNormal()
	r.SetSensitive(true)
	r.SetTooltipText("")
}

// SetSelected is used for highlighting the current message server.
func (r *ServerRow) SetSelected(selected bool) {
	AssertUnhollow(r)

	r.Button.SetSelected(selected)
}

func (r *ServerRow) GetActive() bool {
	if !r.IsHollow() {
		return r.Button.GetActive()
	}

	return false
}

// SetRevealChild reveals the list of servers. It does nothing if there are no
// servers, meaning if Row does not represent a ServerList.
func (r *ServerRow) SetRevealChild(reveal bool) {
	AssertUnhollow(r)

	// Do the above noop check.
	if r.children == nil {
		return
	}

	// Actually reveal the children.
	r.childrev.SetRevealChild(reveal)

	// Save the path.
	savepath.Update(r, reveal)

	if reveal {
		primitives.AddClass(r, "expanded")
	} else {
		primitives.RemoveClass(r, "expanded")
	}

	// If this isn't a reveal, then we don't need to load.
	if !reveal {
		return
	}

	// Ensure that we have successfully loaded the server.
	r.load(func(err error) {
		if err == nil {
			// Load the list of servers if we're still in loading mode. Before,
			// we have to call Servers on this. Now, we already know that there
			// are hollow servers in the children container.
			r.children.LoadAll()
		}
	})
}

// GetRevealChild returns whether or not the server list is expanded, or always
// false if there is no server list.
func (r *ServerRow) GetRevealChild() bool {
	AssertUnhollow(r)

	if r.childrev != nil {
		return r.childrev.GetRevealChild()
	}
	return false
}
