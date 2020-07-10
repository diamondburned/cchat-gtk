package message

import (
	"fmt"
	"html"

	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/attachment"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var EmptyContentPlaceholder = fmt.Sprintf(
	`<span alpha="25%%">%s</span>`, html.EscapeString("<empty>"),
)

type PresendContainer interface {
	SetDone(id string)
	SetLoading()
	SetSentError(err error)
}

// PresendGenericContainer is the generic container with extra methods
// implemented for stateful mutability of the generic message container.
type GenericPresendContainer struct {
	*GenericContainer

	// states; to be cleared on SetDone()
	presend input.PresendMessage
	uploads *attachment.MessageUploader
}

var _ PresendContainer = (*GenericPresendContainer)(nil)

func NewPresendContainer(msg input.PresendMessage) *GenericPresendContainer {
	return WrapPresendContainer(NewEmptyContainer(), msg)
}

func WrapPresendContainer(c *GenericContainer, msg input.PresendMessage) *GenericPresendContainer {
	c.nonce = msg.Nonce()
	c.authorID = msg.AuthorID()
	c.UpdateTimestamp(msg.Time())
	c.UpdateAuthorName(msg.Author())

	p := &GenericPresendContainer{
		GenericContainer: c,

		presend: msg,
		uploads: attachment.NewMessageUploader(msg.Files()),
	}
	p.SetLoading()

	return p
}

func (m *GenericPresendContainer) SetSensitive(sensitive bool) {
	m.contentBox.SetSensitive(sensitive)
}

func (m *GenericPresendContainer) SetDone(id string) {
	// Apply the received ID.
	m.id = id
	// Set the sensitivity from false in SetLoading back to true.
	m.SetSensitive(true)
	// Reset the state to be normal. Especially setting presend to nil should
	// free it from memory.
	m.presend = nil
	m.uploads = nil
	m.contentBox.SetTooltipText("")

	// Remove everything in the content box.
	m.clearBox()

	// Re-add the content label.
	m.contentBox.Add(m.ContentBody)
}

func (m *GenericPresendContainer) SetLoading() {
	m.SetSensitive(false)
	m.contentBox.SetTooltipText("")

	// Clear everything inside the content container.
	m.clearBox()

	// Add the content label.
	m.contentBox.Add(m.ContentBody)

	// Add the attachment progress box back in, if any.
	if m.uploads != nil {
		m.uploads.Show() // show the bars
		m.contentBox.Add(m.uploads)
	}

	if content := m.presend.Content(); content != "" {
		m.ContentBody.SetText(content)
	} else {
		// Use a placeholder content if the actual content is empty.
		m.ContentBody.SetMarkup(EmptyContentPlaceholder)
	}
}

func (m *GenericPresendContainer) SetSentError(err error) {
	m.SetSensitive(true) // allow events incl right clicks
	m.contentBox.SetTooltipText(err.Error())

	// Remove everything again.
	m.clearBox()

	// Re-add the label.
	m.contentBox.Add(m.ContentBody)

	// Style the label appropriately by making it red.
	var content = html.EscapeString(m.presend.Content())
	if content == "" {
		content = EmptyContentPlaceholder
	}
	m.ContentBody.SetMarkup(fmt.Sprintf(`<span color="red">%s</span>`, content))

	// Add a smaller label indicating an error.
	errl, _ := gtk.LabelNew("")
	errl.SetXAlign(0)
	errl.SetLineWrap(true)
	errl.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	errl.SetMarkup(fmt.Sprintf(
		`<span size="small" color="red"><b>Error:</b> %s</span>`,
		html.EscapeString(humanize.Error(err)),
	))

	errl.Show()
	m.contentBox.Add(errl)
}

// clearBox clears everything inside the content container.
func (m *GenericPresendContainer) clearBox() {
	m.contentBox.GetChildren().Foreach(func(v interface{}) {
		m.contentBox.Remove(v.(gtk.IWidget))
	})
}
