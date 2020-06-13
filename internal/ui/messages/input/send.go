package input

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

var globalID uint64

// SendInput yanks the text from the input field and sends it to the backend.
// This function is not thread-safe.
func (f *Field) SendInput() {
	if f.Sender == nil {
		return
	}

	var text = f.yankText()
	if text == "" {
		return
	}

	f.SendMessage(SendMessageData{
		time:      time.Now(),
		content:   text,
		author:    f.username.GetLabel(),
		authorID:  f.UserID,
		authorURL: f.username.GetIconURL(),
		nonce:     "__cchat-gtk_" + strconv.FormatUint(atomic.AddUint64(&globalID, 1), 10),
	})
}

func (f *Field) SendMessage(data PresendMessage) {
	// presend message into the container through the controller
	var onErr = f.ctrl.AddPresendMessage(data)

	go func(sender cchat.ServerMessageSender) {
		if err := sender.SendMessage(data); err != nil {
			gts.ExecAsync(func() { onErr(err) })
			log.Error(errors.Wrap(err, "Failed to send message"))
		}
	}(f.Sender)
}

type SendMessageData struct {
	time      time.Time
	content   string
	author    text.Rich
	authorID  string
	authorURL string // avatar
	nonce     string
}

type PresendMessage interface {
	cchat.MessageHeader // returns nonce and time
	cchat.SendableMessage
	cchat.MessageNonce

	Author() text.Rich
	AuthorID() string
	AuthorAvatarURL() string // may be empty
}

var _ PresendMessage = (*SendMessageData)(nil)

// ID returns a pseudo ID for internal use.
func (s SendMessageData) ID() string {
	return s.nonce
}

func (s SendMessageData) Time() time.Time {
	return s.time
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

func (s SendMessageData) AuthorAvatarURL() string {
	return s.authorURL
}

func (s SendMessageData) Nonce() string {
	return s.nonce
}
