package message

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Container interface {
	ID() cchat.ID
	Time() time.Time
	Author() cchat.Author
	Nonce() string

	UpdateAuthor(cchat.Author)
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
func RefreshContainer(c Container, gc *GenericContainer) {
	c.UpdateTimestamp(gc.time)
}

// GenericContainer provides a single generic message container for subpackages
// to use.
type GenericContainer struct {
	*gtk.Box
	row   *gtk.ListBoxRow // contains Box
	class string

	id     string
	time   time.Time
	author Author
	nonce  string

	Content          *gtk.Box
	ContentBody      *labeluri.Label
	ContentBodyStyle *gtk.StyleContext

	menuItems []menu.Item
}

var _ Container = (*GenericContainer)(nil)

// NewContainer creates a new message container with the given ID and nonce. It
// does not update the widgets, so FillContainer should be called afterwards.
func NewContainer(msg cchat.MessageCreate) *GenericContainer {
	c := NewEmptyContainer()
	c.id = msg.ID()
	c.time = msg.Time()
	c.nonce = msg.Nonce()
	c.author.Update(msg.Author())

	return c
}

func NewEmptyContainer() *GenericContainer {
	ctbody := labeluri.NewLabel(text.Rich{})
	ctbody.SetVExpand(true)
	ctbody.SetHAlign(gtk.ALIGN_START)
	ctbody.SetEllipsize(pango.ELLIPSIZE_NONE)
	ctbody.SetLineWrap(true)
	ctbody.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	ctbody.SetXAlign(0) // left align
	ctbody.SetSelectable(true)
	ctbody.SetTrackVisitedLinks(false)
	ctbody.Show()

	ctbodyStyle, _ := ctbody.GetStyleContext()
	ctbodyStyle.AddClass("message-content")

	// Wrap the content label inside a content box.
	ctbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	ctbox.SetHExpand(true)
	ctbox.PackStart(ctbody, false, false, 0)
	ctbox.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()

	row, _ := gtk.ListBoxRowNew()
	row.Add(box)
	row.Show()
	primitives.AddClass(row, "message-row")

	gc := &GenericContainer{
		Box: box,
		row: row,

		Content:          ctbox,
		ContentBody:      ctbody,
		ContentBodyStyle: ctbodyStyle,

		// Time is important, as it is used to sort messages, so we have to be
		// careful with this.
		time: time.Now(),
	}

	// Bind the custom popup menu to the content label.
	gc.ContentBody.Connect("populate-popup", func(l *gtk.Label, m *gtk.Menu) {
		menu.MenuSeparator(m)
		menu.MenuItems(m, gc.menuItems)
	})

	return gc
}

// Row returns the internal list box row. It is used to satisfy MessageRow.
func (m *GenericContainer) Row() *gtk.ListBoxRow { return m.row }

// SetClass sets the internal row's class.
func (m *GenericContainer) SetClass(class string) {
	if m.class != "" {
		primitives.RemoveClass(m.row, m.class)
	}

	primitives.AddClass(m.row, class)
	m.class = class
}

// SetReferenceHighlighter sets the reference highlighter into the message.
func (m *GenericContainer) SetReferenceHighlighter(r labeluri.ReferenceHighlighter) {
	m.ContentBody.SetReferenceHighlighter(r)
}

func (m *GenericContainer) ID() string {
	return m.id
}

func (m *GenericContainer) Time() time.Time {
	return m.time
}

func (m *GenericContainer) Author() cchat.Author {
	return m.author
}

func (m *GenericContainer) Nonce() string {
	return m.nonce
}

func (m *GenericContainer) UpdateTimestamp(t time.Time) {
	m.time = t
}

func (m *GenericContainer) UpdateAuthor(author cchat.Author) {
	m.author.Update(author)
}

func (m *GenericContainer) UpdateContent(content text.Rich, edited bool) {
	m.ContentBody.SetLabelUnsafe(content)

	if edited {
		markup := m.ContentBody.Output().Markup
		markup += " " + rich.Small("(edited)")
		m.ContentBody.SetMarkup(markup)
	}
}

// AttachMenu connects signal handlers to handle a list of menu items from
// the container.
func (m *GenericContainer) AttachMenu(newItems []menu.Item) {
	m.menuItems = newItems
}

// MenuItems returns the list of menu items for this message.
func (m *GenericContainer) MenuItems() []menu.Item {
	return m.menuItems
}

func (m *GenericContainer) Focusable() gtk.IWidget {
	return m.Content
}
