package message

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

type Container interface {
	ID() string
	AuthorID() string
	AvatarURL() string // avatar
	Nonce() string

	UpdateAuthor(cchat.MessageAuthor)
	UpdateAuthorName(text.Rich)
	UpdateContent(c text.Rich, edited bool)
	UpdateTimestamp(time.Time)
}

// FillContainer sets the container's contents to the one from MessageCreate.
func FillContainer(c Container, msg cchat.MessageCreate) {
	c.UpdateAuthor(msg.Author())
	c.UpdateContent(msg.Content(), false)
	c.UpdateTimestamp(msg.Time())
}

// RefreshContainer sets the container's contents to the one from
// GenericContainer. This is mainly used for transferring between different
// containers.
//
// Right now, this only works with Timestamp, as that's the only state tracked.
func RefreshContainer(c Container, gc *GenericContainer) {
	c.UpdateTimestamp(gc.time)
}

// GenericContainer provides a single generic message container for subpackages
// to use.
type GenericContainer struct {
	id        string
	time      time.Time
	authorID  string
	avatarURL string // avatar
	nonce     string

	Timestamp *gtk.Label
	Username  *gtk.Label
	Content   *gtk.TextView
	CBuffer   *gtk.TextBuffer

	MenuItems []menu.Item
}

var _ Container = (*GenericContainer)(nil)

// NewContainer creates a new message container with the given ID and nonce. It
// does not update the widgets, so FillContainer should be called afterwards.
func NewContainer(msg cchat.MessageCreate) *GenericContainer {
	c := NewEmptyContainer()
	c.id = msg.ID()
	c.time = msg.Time()
	c.authorID = msg.Author().ID()

	if noncer, ok := msg.(cchat.MessageNonce); ok {
		c.nonce = noncer.Nonce()
	}

	return c
}

func NewEmptyContainer() *GenericContainer {
	ts, _ := gtk.LabelNew("")
	ts.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
	ts.SetXAlign(1) // right align
	ts.SetVAlign(gtk.ALIGN_END)
	ts.SetSelectable(true)
	ts.Show()

	user, _ := gtk.LabelNew("")
	user.SetMaxWidthChars(35)
	user.SetLineWrap(true)
	user.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	user.SetHAlign(gtk.ALIGN_END)
	user.SetVAlign(gtk.ALIGN_START)
	user.SetSelectable(true)
	user.Show()

	content, _ := gtk.TextViewNew()
	content.SetHExpand(true)
	content.SetWrapMode(gtk.WRAP_WORD_CHAR)
	content.SetCursorVisible(false)
	content.SetEditable(false)
	content.Show()

	cbuffer, _ := content.GetBuffer()

	// Add CSS classes.
	primitives.AddClass(ts, "message-time")
	primitives.AddClass(user, "message-author")
	primitives.AddClass(content, "message-content")

	gc := &GenericContainer{
		Timestamp: ts,
		Username:  user,
		Content:   content,
		CBuffer:   cbuffer,
	}

	gc.Content.SetProperty("populate-all", true)
	gc.Content.Connect("populate-popup", func(tv *gtk.TextView, popup *gtk.Widget) {
		v, err := popup.Cast()
		if err != nil {
			log.Error(errors.Wrap(err, "Failed to cast popup to IWidget"))
			return
		}

		switch popup := v.(type) {
		case menu.MenuAppender:
			menu.MenuSeparator(popup)
			menu.MenuItems(popup, gc.MenuItems)

		case menu.ToolbarInserter:
			menu.ToolbarSeparator(popup)
			menu.ToolbarItems(popup, gc.MenuItems)

		default:
			log.Printlnf("Debug: typeOf(popup) = %T", popup)
		}
	})

	return gc
}

func (m *GenericContainer) ID() string {
	return m.id
}

func (m *GenericContainer) Time() time.Time {
	return m.time
}

func (m *GenericContainer) AuthorID() string {
	return m.authorID
}

func (m *GenericContainer) AvatarURL() string {
	return m.avatarURL
}

func (m *GenericContainer) Nonce() string {
	return m.nonce
}

func (m *GenericContainer) UpdateTimestamp(t time.Time) {
	m.time = t
	m.Timestamp.SetMarkup(rich.Small(humanize.TimeAgo(t)))
	m.Timestamp.SetTooltipText(t.Format(time.Stamp))
}

func (m *GenericContainer) UpdateAuthor(author cchat.MessageAuthor) {
	m.authorID = author.ID()
	m.UpdateAuthorName(author.Name())

	// Set the avatar URL for future access on-demand.
	if avatarer, ok := author.(cchat.MessageAuthorAvatar); ok {
		m.avatarURL = avatarer.Avatar()
	}
}

func (m *GenericContainer) UpdateAuthorName(name text.Rich) {
	m.Username.SetMarkup(parser.RenderMarkup(name))
}

func (m *GenericContainer) UpdateContent(content text.Rich, edited bool) {
	// Render the content.
	parser.RenderTextBuffer(m.CBuffer, content)

	if edited {
		parser.AppendEditBadge(m.CBuffer, m.Time())
	}
}

// AttachMenu connects signal handlers to handle a list of menu items from
// the container.
func (m *GenericContainer) AttachMenu(newItems []menu.Item) {
	m.MenuItems = newItems
}
