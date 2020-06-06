package message

import (
	"html"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat/text"
)

type PresendContainer interface {
	Container
	SetID(id string)
	SetDone()
	SetSentError(err error)
}

// PresendGenericContainer is the generic container with extra methods
// implemented for mutability of the generic message container.
type GenericPresendContainer struct {
	*GenericContainer
}

var _ PresendContainer = (*GenericPresendContainer)(nil)

func NewPresendContainer(msg input.PresendMessage) *GenericPresendContainer {
	c := NewEmptyContainer()
	c.nonce = msg.Nonce()
	c.authorID = msg.AuthorID()
	c.UpdateContent(text.Rich{Content: msg.Content()})
	c.UpdateTimestamp(time.Now())
	c.UpdateAuthorName(msg.Author())

	p := &GenericPresendContainer{
		GenericContainer: c,
	}
	p.SetSensitive(false)

	return p
}

func (m *GenericPresendContainer) SetID(id string) {
	m.id = id
}

func (m *GenericPresendContainer) SetSensitive(sensitive bool) {
	m.Content.SetSensitive(sensitive)
}

func (m *GenericPresendContainer) SetDone() {
	m.SetSensitive(true)
}

func (m *GenericPresendContainer) SetSentError(err error) {
	m.Content.SetMarkup(`<span color="red">` + html.EscapeString(m.Content.GetLabel()) + `</span>`)
	m.Content.SetTooltipText(err.Error())
}
