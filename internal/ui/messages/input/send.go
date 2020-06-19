package input

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
	"github.com/twmb/murmur3"
)

var globalID uint64

// generateNonce creates a nonce that should prevent collision. This function
// will always return a 24-byte long string.
func (f *Field) generateNonce() string {
	raw := fmt.Sprintf(
		"cchat-gtk/%s/%X/%X",
		f.UserID, time.Now().UnixNano(), atomic.AddUint64(&globalID, 1),
	)

	h1, h2 := murmur3.StringSum128(raw)
	nonce := make([]byte, 8*2)
	binary.LittleEndian.PutUint64(nonce[0:8], h1)
	binary.LittleEndian.PutUint64(nonce[8:16], h2)

	return base64.RawURLEncoding.EncodeToString(nonce)
}

func (f *Field) sendInput() {
	if f.Sender == nil {
		return
	}

	var text = f.yankText()
	if text == "" {
		return
	}

	// Are we editing anything?
	if id := f.editingID; f.Editable() && id != "" {
		go func() {
			if err := f.editor.EditMessage(id, text); err != nil {
				log.Error(errors.Wrap(err, "Failed to edit message"))
			}
		}()

		f.StopEditing()
		return
	}

	f.SendMessage(SendMessageData{
		time:      time.Now(),
		content:   text,
		author:    f.username.GetLabel(),
		authorID:  f.UserID,
		authorURL: f.username.GetIconURL(),
		nonce:     f.generateNonce(),
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
