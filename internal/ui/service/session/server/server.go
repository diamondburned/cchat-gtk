package server

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/breadcrumb"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/button"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/loading"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const ChildrenMargin = 24
const IconSize = 20

type Controller interface {
	RowSelected(*ServerRow, cchat.ServerMessage)
}

type Row struct {
	*gtk.Box
	Button *button.ToggleButtonImage

	parentcrumb breadcrumb.Breadcrumber

	children   *Children
	serverList cchat.ServerList
	loaded     bool
}

func NewRow(parent breadcrumb.Breadcrumber, name text.Rich) *Row {
	button := button.NewToggleButtonImage(name)
	button.Box.SetHAlign(gtk.ALIGN_START)
	button.Image.AddProcessors(imgutil.Round(true))
	button.Image.SetSize(IconSize)
	button.SetRelief(gtk.RELIEF_NONE)
	button.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.SetMarginStart(ChildrenMargin)
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

	r.Box.PackStart(r.children, false, false, 0)
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
	r.Button.SetLoading()
	r.SetSensitive(false)
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
	r.SetDone()
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
	r.children.SetRevealChild(reveal)

	// If this isn't a reveal, then we don't need to load.
	if !reveal {
		return
	}

	// If we haven't loaded yet and we're still not loading, then load.
	if !r.loaded && r.children.load == nil {
		r.Load()
	}
}

// Load loads the row without uncollapsing it.
func (r *Row) Load() {
	// Safeguard.
	if r.children == nil || r.serverList == nil {
		return
	}

	// Set that we're now loading.
	r.children.setLoading()
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

// GetRevealChild returns whether or not the server list is expanded, or always
// false if there is no server list.
func (r *Row) GetRevealChild() bool {
	if r.children != nil {
		return r.children.GetRevealChild()
	}
	return false
}

type ServerRow struct {
	*Row
	Server cchat.Server
}

func NewServerRow(p breadcrumb.Breadcrumber, server cchat.Server, ctrl Controller) *ServerRow {
	row := NewRow(p, server.Name())
	row.Show()
	row.SetIconer(server)
	primitives.AddClass(row, "server")

	var serverRow = &ServerRow{Row: row, Server: server}

	switch server := server.(type) {
	case cchat.ServerList:
		row.SetServerList(server, ctrl)
		primitives.AddClass(row, "server-list")

	case cchat.ServerMessage:
		row.Button.SetClickedIfTrue(func() { ctrl.RowSelected(serverRow, server) })
		primitives.AddClass(row, "server-message")
	}

	return serverRow
}

// Children is a children server with a reference to the parent.
type Children struct {
	*gtk.Revealer
	Main *gtk.Box

	rowctrl Controller

	load *loading.Button // only not nil while loading

	Rows   []*ServerRow
	Parent breadcrumb.Breadcrumber
}

func NewChildren(p breadcrumb.Breadcrumber, ctrl Controller) *Children {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.Add(main)

	return &Children{
		Revealer: rev,
		Main:     main,
		rowctrl:  ctrl,
		Parent:   p,
	}
}

// setLoading shows the loading circle as a list child.
func (c *Children) setLoading() {
	// Exit if we're already loading.
	if c.load != nil {
		return
	}

	// Clear everything.
	c.Reset()

	// Set the loading circle and stuff.
	c.load = loading.NewButton()
	c.load.Show()
	c.Main.Add(c.load)
}

func (c *Children) Reset() {
	// Remove old servers from the list.
	for _, row := range c.Rows {
		c.Main.Remove(row)
	}

	// Wipe the list empty.
	c.Rows = nil
}

// setNotLoading removes the loading circle, if any. This is not in Reset()
// anymore, since the backend may not necessarily call SetServers.
func (c *Children) setNotLoading() {
	// Do we have the spinning circle button? If yes, remove it.
	if c.load != nil {
		// Stop the loading mode. The reset function should do everything for us.
		c.Main.Remove(c.load)
		c.load = nil
	}
}

func (c *Children) SetServers(servers []cchat.Server) {
	gts.ExecAsync(func() {
		// Save the current state.
		var oldID string
		for _, row := range c.Rows {
			if row.GetActive() {
				oldID = row.Server.ID()
				break
			}
		}

		// Reset before inserting new servers.
		c.Reset()

		c.Rows = make([]*ServerRow, len(servers))

		for i, server := range servers {
			row := NewServerRow(c, server, c.rowctrl)
			c.Rows[i] = row
			c.Main.Add(row)
		}

		// Update parent reference? Only if it's activated.
		if oldID != "" {
			for _, row := range c.Rows {
				if row.Server.ID() == oldID {
					row.Button.SetActive(true)
				}
			}
		}
	})
}

func (c *Children) Breadcrumb() breadcrumb.Breadcrumb {
	return breadcrumb.Try(c.Parent)
}
