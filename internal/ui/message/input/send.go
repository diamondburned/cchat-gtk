package input

import (
	"strconv"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

type SendMessageData struct {
	content  string
	author   text.Rich
	authorID string
	nonce    string
}

type PresendMessage interface {
	cchat.SendableMessage
	cchat.MessageNonce

	Author() text.Rich
	AuthorID() string
}

var (
	_ cchat.SendableMessage = (*SendMessageData)(nil)
	_ cchat.MessageNonce    = (*SendMessageData)(nil)
)

func NewSendMessageData(content string, author text.Rich, authorID string) SendMessageData {
	return SendMessageData{
		content: content,
		author:  author,
		nonce:   "cchat-gtk_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
}

func (s SendMessageData) Content() string {
	return s.content
}

func (s SendMessageData) Author() text.Rich {
	return s.author
}

func (s SendMessageData) AuthorID() string {
	return s.authorID
}

func (s SendMessageData) Nonce() string {
	return s.nonce
}
