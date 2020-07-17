package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/button"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 24
const IconSize = 32

type ServerRow struct {
	*Row
	Server cchat.Server
}

var serverCSS = primitives.PrepareClassCSS("server", `
	/* Ignore first child because .server-children already covers this */
	.server:not(:first-child) {
		margin: 0;
		margin-top: 3px;
		border-radius: 0;
	}
`)

func NewServerRow(p breadcrumb.Breadcrumber, server cchat.Server, ctrl Controller) *ServerRow {
	row := NewRow(p, server.Name())
	row.SetIconer(server)
	serverCSS(row)

	var serverRow = &ServerRow{Row: row, Server: server}

	switch server := server.(type) {
	case cchat.ServerList:
		row.SetServerList(server, ctrl)
		primitives.AddClass(row, "server-list")

	case cchat.ServerMessage:
		row.Button.SetClickedIfTrue(func() { ctrl.RowSelected(serverRow, server) })
		primitives.AddClass(row, "server-message")

		// Check if the server is capable of indicating unread state.
		if unreader, ok := server.(cchat.ServerMessageUnreadIndicator); ok {
			// Set as read by default.
			row.Button.SetUnreadUnsafe(false, false)

			gts.Async(func() (func(), error) {
				c, err := unreader.UnreadIndicate(row)
				if err != nil {
					return nil, errors.Wrap(err, "Failed to use unread indicator")
				}

				return func() { row.Connect("destroy", c) }, nil
			})
		}
	}

	return serverRow
}

type Row struct {
	*gtk.Box
	Button *button.ToggleButtonImage

	parentcrumb breadcrumb.Breadcrumber

	childrev   *gtk.Revealer
	children   *Children
	serverList cchat.ServerList
	loaded     bool
}

func NewRow(parent breadcrumb.Breadcrumber, name text.Rich) *Row {
	button := button.NewToggleButtonImage(name)
	button.Box.SetHAlign(gtk.ALIGN_START)
	button.SetRelief(gtk.RELIEF_NONE)
	button.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(button, false, false, 0)

	row := &Row{
		Box:         box,
		Button:      button,
		parentcrumb: parent,
	}

	return row
}

func (r *Row) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(r.parentcrumb, r.Button.GetText())
}

func (r *Row) SetLabelUnsafe(name text.Rich) {
	r.Button.SetLabelUnsafe(name)
}

func (r *Row) SetIconer(v interface{}) {
	if iconer, ok := v.(cchat.Icon); ok {
		r.Button.Image.SetSize(IconSize)
		r.Button.Image.AsyncSetIconer(iconer, "Error getting server icon URL")
	}
}

// SetServerList sets the row to a server list.
func (r *Row) SetServerList(list cchat.ServerList, ctrl Controller) {
	r.Button.SetClicked(func(active bool) {
		r.SetRevealChild(active)
	})

	r.children = NewChildren(r, ctrl)
	r.children.Show()

	r.childrev, _ = gtk.RevealerNew()
	r.childrev.SetRevealChild(false)
	r.childrev.Add(r.children)
	r.childrev.Show()

	r.Box.PackStart(r.childrev, false, false, 0)
	r.serverList = list
}

// Reset clears off all children servers. It's a no-op if there are none.
func (r *Row) Reset() {
	if r.children != nil {
		// Remove everything from the children container.
		r.children.Reset()

		// Remove the children container itself.
		r.Box.Remove(r.children)
	}

	// Reset the state.
	r.loaded = false
	r.serverList = nil
	r.children = nil
}

// SetLoading is called by the parent struct.
func (r *Row) SetLoading() {
	r.SetSensitive(false)
	r.Button.SetLoading()
}

// SetFailed is shared between the parent struct and the children list. This is
// because both of those errors share the same appearance, just different
// callbacks.
func (r *Row) SetFailed(err error, retry func()) {
	r.SetSensitive(true)
	r.SetTooltipText(err.Error())
	r.Button.SetFailed(err, retry)
	r.Button.Label.SetMarkup(rich.MakeRed(r.Button.GetLabel()))
}

// SetDone is shared between the parent struct and the children list. This is
// because both will use the same SetFailed.
func (r *Row) SetDone() {
	r.Button.SetNormal()
	r.SetSensitive(true)
	r.SetTooltipText("")
}

func (r *Row) SetNormalExtraMenu(items []menu.Item) {
	r.Button.SetNormalExtraMenu(items)
	r.SetSensitive(true)
	r.SetTooltipText("")
}

func (r *Row) childrenFailed(err error) {
	// If the user chooses to retry, the list will automatically expand.
	r.SetFailed(err, func() { r.SetRevealChild(true) })
}

func (r *Row) childrenDone() {
	r.loaded = true

	// I don't think this is supposed to be called here...
	// r.SetDone()
}

// SetSelected is used for highlighting the current message server.
func (r *Row) SetSelected(selected bool) {
	// Set the clickability the opposite as the boolean.
	r.Button.SetSensitive(!selected)

	// Some special edge case that I forgot.
	if !selected {
		r.Button.SetActive(false)
	}
}

func (r *Row) GetActive() bool {
	return r.Button.GetActive()
}

// SetRevealChild reveals the list of servers. It does nothing if there are no
// servers, meaning if Row does not represent a ServerList.
func (r *Row) SetRevealChild(reveal bool) {
	// Do the above noop check.
	if r.children == nil {
		return
	}

	// Actually reveal the children.
	r.childrev.SetRevealChild(reveal)

	// If this isn't a reveal, then we don't need to load.
	if !reveal {
		return
	}

	// If we haven't loaded yet and we're still not loading, then load.
	if !r.loaded && r.children.load == nil {
		r.Load()
	}
}

// GetRevealChild returns whether or not the server list is expanded, or always
// false if there is no server list.
func (r *Row) GetRevealChild() bool {
	if r.childrev != nil {
		return r.childrev.GetRevealChild()
	}
	return false
}

// Load loads the row without uncollapsing it.
func (r *Row) Load() {
	// Safeguard.
	if r.children == nil || r.serverList == nil {
		return
	}

	// Set that we're now loading.
	r.children.setLoading()
	r.SetLoading()
	r.SetSensitive(false)

	// Load the list of servers if we're still in loading mode.
	go func() {
		err := r.serverList.Servers(r.children)
		gts.ExecAsync(func() {
			// We're not loading anymore, so remove the loading circle.
			r.children.setNotLoading()
			// Restore clickability.
			r.SetSensitive(true)

			// Use the childrenX method instead of SetX.
			if err != nil {
				r.childrenFailed(errors.Wrap(err, "Failed to get servers"))
			} else {
				r.childrenDone()
			}
		})
	}()
}

// SetUnread is thread-safe.
func (r *Row) SetUnread(unread, mentioned bool) {
	gts.ExecAsync(func() { r.SetUnreadUnsafe(unread, mentioned) })
}

func (r *Row) SetUnreadUnsafe(unread, mentioned bool) {
	r.Button.SetUnreadUnsafe(unread, mentioned)
}
