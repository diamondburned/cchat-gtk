package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/actions"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/savepath"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/button"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/commander"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 24
const IconSize = 32

func AssertUnhollow(hollower interface{ IsHollow() bool }) {
	if hollower.IsHollow() {
		panic("Server is hollow, but a normal method was called.")
	}
}

type ServerRow struct {
	*gtk.Box
	Avatar      *roundimage.Avatar
	Button      *button.ToggleButtonImage
	ActionsMenu *actions.Menu

	Server cchat.Server
	ctrl   Controller

	parentcrumb traverse.Breadcrumber

	cmder *commander.Buffer

	// non-nil if server list and the function returns error
	childrenErr error

	childrev   *gtk.Revealer
	children   *Children
	serverList cchat.Lister

	// State that's updated even when stale. Initializations will use these.
	unread    bool
	mentioned bool

	// callback to cancel unread indicator
	cancelUnread func()
}

var serverCSS = primitives.PrepareClassCSS("server", `
	/* Ignore first child because .server-children already covers this */
	.server:not(:first-child) {
		margin: 0;
		margin-top: 3px;
		border-radius: 0;
	}
`)

// NewHollowServer creates a new hollow ServerRow. It will automatically create
// hollow children containers and rows for the given server.
func NewHollowServer(p traverse.Breadcrumber, sv cchat.Server, ctrl Controller) *ServerRow {
	var serverRow = &ServerRow{
		parentcrumb:  p,
		ctrl:         ctrl,
		Server:       sv,
		cancelUnread: func() {},
	}

	var (
		lister    = sv.AsLister()
		messenger = sv.AsMessenger()
	)

	switch {
	case lister != nil:
		serverRow.SetHollowServerList(lister, ctrl)
		serverRow.children.SetUnreadHandler(serverRow.SetUnreadUnsafe)

	case messenger != nil:
		if unreader := messenger.AsUnreadIndicator(); unreader != nil {
			gts.Async(func() (func(), error) {
				c, err := unreader.UnreadIndicate(serverRow)
				if err != nil {
					return nil, errors.Wrap(err, "Failed to use unread indicator")
				}

				return func() { serverRow.cancelUnread = c }, nil
			})
		}
	}

	return serverRow
}

// Init brings the row out of the hollow state. It loads the children (if any),
// but this process does not make more widgets.
func (r *ServerRow) Init() {
	if !r.IsHollow() {
		return
	}

	// Initialize the row, which would fill up the button and others as well.
	r.Avatar = roundimage.NewAvatar(IconSize)
	r.Avatar.SetText(r.Server.Name().Content)
	r.Avatar.Show()

	btn := rich.NewCustomToggleButtonImage(r.Avatar, r.Server.Name())
	btn.Show()

	r.Button = button.WrapToggleButtonImage(btn)
	r.Button.Box.SetHAlign(gtk.ALIGN_START)
	r.Button.SetRelief(gtk.RELIEF_NONE)
	r.Button.Show()

	r.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	r.Box.PackStart(r.Button, false, false, 0)
	serverCSS(r.Box)

	// Ensure errors are displayed.
	r.childrenSetErr(r.childrenErr)

	// Try to set an icon.
	r.SetIconer(r.Server)

	// Connect the destroyer, if any.
	r.Connect("destroy", r.cancelUnread)

	// Restore the read state.
	r.Button.SetUnreadUnsafe(r.unread, r.mentioned) // update with state

	// Make the Actions menu.
	r.ActionsMenu = actions.NewMenu("server")
	r.ActionsMenu.InsertActionGroup(r)

	if cmder := r.Server.AsCommander(); cmder != nil {
		r.cmder = commander.NewBuffer(r.Server.Name().String(), cmder)
		r.ActionsMenu.AddAction("Command Prompt", r.cmder.ShowDialog)
	}

	// Bind right clicks and show a popover menu on such event.
	r.Button.Connect("button-press-event", func(_ gtk.IWidget, ev *gdk.Event) {
		if gts.EventIsRightClick(ev) {
			r.ActionsMenu.Popover(r).Popup()
		}
	})

	var (
		lister    = r.Server.AsLister()
		messenger = r.Server.AsMessenger()
	)

	switch {
	case lister != nil:
		primitives.AddClass(r, "server-list")
		r.children.Init()
		r.children.Show()

		r.childrev, _ = gtk.RevealerNew()
		r.childrev.SetRevealChild(false)
		r.childrev.Add(r.children)
		r.childrev.Show()

		r.Box.PackStart(r.childrev, false, false, 0)
		r.Button.SetClicked(r.SetRevealChild)

	case messenger != nil:
		primitives.AddClass(r, "server-message")
		r.Button.SetClicked(func(bool) { r.ctrl.MessengerSelected(r) })
	}
}

// GetActiveServerMessage returns true if the row is currently selected AND it
// is a message row.
func (r *ServerRow) GetActiveServerMessage() bool {
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
	if r.GetActiveServerMessage() {
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
	r.children.setLoading()

	go func() {
		var err = list.Servers(r.children)
		if err != nil {
			log.Error(errors.Wrap(err, "Failed to get servers"))
		}

		gts.ExecAsync(func() {
			// Announce that we're not loading anymore.
			r.children.setNotLoading()

			if !r.IsHollow() {
				// Restore clickability.
				r.SetSensitive(true)
			}

			// Use the childrenX method instead of SetX. We can wrap nil
			// errors.
			r.childrenSetErr(errors.Wrap(err, "Failed to get servers"))
		})
	}()
}

// Reset clears off all children servers. It's a no-op if there are none.
func (r *ServerRow) Reset() {
	if r.children != nil {
		// Remove everything from the children container.
		r.children.Reset()

		// Remove the children container itself.
		r.Box.Remove(r.children)
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

	r.Button.Image.SetSize(IconSize)
	r.Button.Image.SetRevealChild(true)
}

// HasIcon returns true if the current row has an icon.
func (r *ServerRow) HasIcon() bool {
	return !r.IsHollow() && r.Button.Image.GetRevealChild()
}

func (r *ServerRow) ParentBreadcrumb() traverse.Breadcrumber {
	return r.parentcrumb
}

func (r *ServerRow) Breadcrumb() string {
	if r.IsHollow() {
		return ""
	}
	return r.Button.GetText()
}

func (r *ServerRow) ID() cchat.ID {
	return r.Server.ID()
}

func (r *ServerRow) SetLabelUnsafe(name text.Rich) {
	AssertUnhollow(r)

	r.Button.SetLabelUnsafe(name)
	r.Avatar.SetText(name.Content)
}

// SetIconer takes in a Namer for AsIconer.
func (r *ServerRow) SetIconer(v cchat.Namer) {
	AssertUnhollow(r)

	if iconer := v.AsIconer(); iconer != nil {
		r.Button.Image.SetSize(IconSize)
		r.Button.Image.AsyncSetIconer(iconer, "Error getting server icon URL")
	}
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
	r.Button.Label.SetMarkup(rich.MakeRed(r.Button.GetLabel()))
}

// SetDone is shared between the parent struct and the children list. This is
// because both will use the same SetFailed.
func (r *ServerRow) SetDone() {
	AssertUnhollow(r)

	r.Button.SetNormal()
	r.SetSensitive(true)
	r.SetTooltipText("")
}

func (r *ServerRow) SetNormalExtraMenu(items []menu.Item) {
	AssertUnhollow(r)

	r.Button.SetNormalExtraMenu(items)
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

	// If this isn't a reveal, then we don't need to load.
	if !reveal {
		return
	}

	// Load the list of servers if we're still in loading mode. Before, we have
	// to call Servers on this. Now, we already know that there are hollow
	// servers in the children container.
	r.children.LoadAll()
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
