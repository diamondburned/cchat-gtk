package input

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

type SendMessageData struct {
	content   string
	author    text.Rich
	authorID  string
	authorURL string // avatar
	nonce     string
}

type PresendMessage interface {
	cchat.SendableMessage
	cchat.MessageNonce

	Author() text.Rich
	AuthorID() string
	AuthorAvatarURL() string // may be empty
}

var _ PresendMessage = (*SendMessageData)(nil)

func (s SendMessageData) Content() string {
	return s.content
}

func (s SendMessageData) Author() text.Rich {
	return s.author
}

func (s SendMessageData) AuthorID() string {
	return s.authorID
}

func (s SendMessageData) AuthorAvatarURL() string {
	return s.authorURL
}

func (s SendMessageData) Nonce() string {
	return s.nonce
}
