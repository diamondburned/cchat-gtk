package message

import (
	"fmt"
	"html"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/attachment"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var EmptyContentPlaceholder = fmt.Sprintf(
	`<span alpha="25%%">%s</span>`, html.EscapeString("<empty>"),
)

// Presender describes actions doable on a presend message container.
type Presender interface {
	SendingMessage() PresendMessage
	SetDone(id cchat.ID)
	SetLoading()
	SetSentError(err error)
}

// PresendMessage is an interface for any message about to be sent.
type PresendMessage interface {
	cchat.MessageHeader
	cchat.SendableMessage
	cchat.Noncer

	// These methods are reserved for internal use.

	Files() []attachment.File
}

// PresendState is the generic state with extra methods implemented for stateful
// mutability of the generic message state.
type PresendState struct {
	*State

	// states; to be cleared on SetDone()
	presend PresendMessage
	uploads *attachment.MessageUploader
}

var (
	_ Presender = (*PresendState)(nil)
)

type SendMessageData struct {
}

// NewPresendState creates a new presend state.
func NewPresendState(self *Author, msg PresendMessage) *PresendState {
	c := NewEmptyState()
	c.Author = self
	c.Nonce = msg.Nonce()
	c.Time = msg.Time()

	p := &PresendState{
		State:   c,
		presend: msg,
		uploads: attachment.NewMessageUploader(msg.Files()),
	}
	p.SetLoading()

	return p
}

func (m *PresendState) SendingMessage() PresendMessage { return m.presend }

// SetSensitive sets the sensitivity of the content.
func (m *PresendState) SetSensitive(sensitive bool) {
	m.Content.SetSensitive(sensitive)
}

// SetDone sets the status of the state.
func (m *PresendState) SetDone(id cchat.ID) {
	// Apply the received ID.
	m.ID = id
	m.Nonce = ""

	// Reset the state to be normal. Especially setting presend to nil should
	// free it from memory.
	m.presend = nil
	m.uploads = nil
	m.Content.SetTooltipText("")

	// Remove everything in the content box.
	m.clearBox()

	// Re-add the content label.
	m.Content.Add(m.ContentBody)

	// Set the sensitivity from false in SetLoading back to true.
	m.SetSensitive(true)
}

// SetLoading greys the message to indicate that it's loading.
func (m *PresendState) SetLoading() {
	m.SetSensitive(false)
	m.Content.SetTooltipText("")

	// Clear everything inside the content container.
	m.clearBox()

	// Add the content label.
	m.Content.Add(m.ContentBody)

	// Add the attachment progress box back in, if any.
	if m.uploads != nil {
		m.uploads.Show() // show the bars
		m.Content.Add(m.uploads)
	}

	if content := m.presend.Content(); content != "" {
		m.ContentBody.SetText(content)
	} else {
		// Use a placeholder content if the actual content is empty.
		m.ContentBody.SetMarkup(EmptyContentPlaceholder)
	}
}

// SetSentError sets the error into the message to notify the user.
func (m *PresendState) SetSentError(err error) {
	m.SetSensitive(true) // allow events incl right clicks
	m.Content.SetTooltipText(err.Error())

	// Remove everything again.
	m.clearBox()

	// Re-add the label.
	m.Content.Add(m.ContentBody)

	// Style the label appropriately by making it red.
	var content = EmptyContentPlaceholder
	if m.presend != nil && m.presend.Content() != "" {
		content = m.presend.Content()
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
	m.Content.Add(errl)
}

// clearBox clears everything inside the content container.
func (m *PresendState) clearBox() {
	primitives.RemoveChildren(m.Content)
}
