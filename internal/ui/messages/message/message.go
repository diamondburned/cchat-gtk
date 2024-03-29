package message

import (
	"context"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const AvatarSize = 40

// Container describes a message container that wraps a state. These methods are
// made for containers to override; methods not meant to be override are not
// exposed and will be done directly on the State.
type Container interface {
	// Unwrap returns the internal message state.
	Unwrap() *State
	// Revert unwraps and reverts all widget changes to the internal state then
	// returns that state.
	Revert() *State

	// UpdateContent updates the underlying content widget.
	UpdateContent(content text.Rich, edited bool)

	// SetReferenceHighlighter sets the reference highlighter into the message.
	SetReferenceHighlighter(refer labeluri.ReferenceHighlighter)
}

// State provides a single generic message container for subpackages
// to use.
type State struct {
	gtk.Box
	Row   *gtk.ListBoxRow // contains Box
	class string

	ID     cchat.ID
	Time   time.Time
	Nonce  string
	Author *Author

	Content          *gtk.Box
	ContentBody      *labeluri.Label
	ContentBodyStyle *gtk.StyleContext

	MenuItems []menu.Item
}

// NewState creates a new message state with the given MessageCreate.
func NewState(msg cchat.MessageCreate) *State {
	author := msg.Author()

	c := NewEmptyState()
	c.Author.ID = author.ID()
	c.Author.Name.QueueNamer(context.Background(), author)
	c.ID = msg.ID()
	c.Time = msg.Time()
	c.Nonce = msg.Nonce()
	c.UpdateContent(msg.Content(), false)

	return c
}

// NewEmptyState creates a new empty message state. The author should be set
// immediately afterwards; it is invalid once the state is used.
func NewEmptyState() *State {
	ctbody := labeluri.NewLabel(text.Rich{})
	ctbody.Tooltip = false
	ctbody.SetHAlign(gtk.ALIGN_FILL)
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
	ctbox.PackStart(ctbody, false, false, 0)
	ctbox.SetHAlign(gtk.ALIGN_FILL)
	ctbox.Show()

	// Box that belongs to the implementations of messages.
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.Show()

	row, _ := gtk.ListBoxRowNew()
	row.Add(box)
	row.Show()
	primitives.AddClass(row, "message-row")

	gc := &State{
		Box:    *box,
		Row:    row,
		Author: &Author{},

		Content:          ctbox,
		ContentBody:      ctbody,
		ContentBodyStyle: ctbodyStyle,

		// Time is important, as it is used to sort messages, so we have to be
		// careful with this.
		Time: time.Now(),
	}

	// This may either work, or it may cause memory leaks.
	row.Connect("destroy", func() { gc.Author.Name.Stop() })

	// Bind the custom popup menu to the content label.
	gc.ContentBody.Connect("populate-popup", func(l *gtk.Label, m *gtk.Menu) {
		menu.MenuSeparator(m)
		menu.MenuItems(m, gc.MenuItems)
	})

	return gc
}

// ClearBox clears the state's widget container.
func (m *State) ClearBox() {
	primitives.RemoveChildren(m)
	m.SetClass("")
}

// // For debugging use only.
// func (m *State) PackStart(child gtk.IWidget, expand bool, fill bool, padding uint) {
// 	paths := make([]string, 0, 5)
// 	for i := 1; i < 5; i++ {
// 		_, file, line, ok := runtime.Caller(i)
// 		if !ok {
// 			break
// 		}
//
// 		paths = append(paths, fmt.Sprintf("%s:%d", filepath.Base(file), line))
// 	}
//
// 	log.Println("child packstart", m.ID, "at", strings.Join(paths, " < "))
// 	m.Box.PackStart(child, expand, fill, padding)
// }

// SetClass sets the internal row's class.
func (m *State) SetClass(class string) {
	if m.class != "" {
		primitives.RemoveClass(m.Row, m.class)
	}

	if class != "" {
		primitives.AddClass(m.Row, class)
	}

	m.class = class
}

// SetReferenceHighlighter sets the reference highlighter into the message.
func (m *State) SetReferenceHighlighter(r labeluri.ReferenceHighlighter) {
	m.ContentBody.SetReferenceHighlighter(r)
}

// UpdateContent replaces the internal content and the widget.
func (m *State) UpdateContent(content text.Rich, edited bool) {
	m.ContentBody.SetLabel(content)

	if edited {
		m.ContentBody.SetRenderer(func(content text.Rich) markup.RenderOutput {
			output := markup.RenderCmplx(content)
			output.Markup += rich.Small(text.Plain("(edited)")).Markup
			return output
		})
	}
}

func (m *State) Focusable() gtk.IWidget {
	return m.Content
}

// Unwrap returns itself.
func (m *State) Unwrap() *State { return m }
