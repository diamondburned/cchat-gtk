package message

import (
	"html"

	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
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
	sendString string // to be cleared on SetDone()
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
		sendString:       msg.Content(),
	}
	p.SetLoading()

	return p
}

func (m *GenericPresendContainer) SetSensitive(sensitive bool) {
	m.Content.SetSensitive(sensitive)
}

func (m *GenericPresendContainer) SetDone(id string) {
	m.id = id
	m.SetSensitive(true)
	m.sendString = ""
	m.Content.SetTooltipText("")
}

func (m *GenericPresendContainer) SetLoading() {
	m.SetSensitive(false)
	m.CBuffer.SetText(m.sendString)
	m.Content.SetTooltipText("")
}

func (m *GenericPresendContainer) SetSentError(err error) {
	m.SetSensitive(true) // allow events incl right clicks
	m.CBuffer.SetText(`<span color="red">` + html.EscapeString(m.sendString) + `</span>`)
	m.Content.SetTooltipText(err.Error())
}
