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
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/attachment"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
	"github.com/twmb/murmur3"
)

// globalID used for atomically generating nonces.
var globalID uint64

// generateNonce creates a nonce that should prevent collision. This function
// will always return a 16-byte long string.
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

	// Get the input text.
	var text = f.getText()

	// Are we editing anything?
	if id := f.editingID; f.Editable(id) && id != "" {
		go func() {
			if err := f.editor.Edit(id, text); err != nil {
				log.Error(errors.Wrap(err, "Failed to edit message"))
			}
		}()

		f.StopEditing()
		return
	}

	// Get the attachments.
	var attachments = f.Attachments.Files()

	// Don't send if the message is empty.
	if text == "" && len(attachments) == 0 {
		return
	}

	f.SendMessage(SendMessageData{
		time:      time.Now().UTC(),
		content:   text,
		author:    f.Username.GetLabel(),
		authorID:  f.UserID,
		authorURL: f.Username.GetIconURL(),
		nonce:     f.generateNonce(),
		files:     attachments,
	})

	// Clear the input field after sending.
	f.clearText()

	// Refocus the textbox.
	f.text.GrabFocus()
}

func (f *Field) SendMessage(data PresendMessage) {
	// presend message into the container through the controller
	var onErr = f.ctrl.AddPresendMessage(data)

	// Copy the sender to prevent race conditions.
	var sender = f.Sender
	gts.Async(func() (func(), error) {
		if err := sender.Send(data); err != nil {
			return func() { onErr(err) }, errors.Wrap(err, "Failed to send message")
		}
		return nil, nil
	})
}

type SendMessageData struct {
	time      time.Time
	content   string
	author    text.Rich
	authorID  string
	authorURL string // avatar
	nonce     string
	files     []attachment.File
}

var _ cchat.SendableMessage = (*SendMessageData)(nil)

type PresendMessage interface {
	cchat.MessageHeader // returns nonce and time
	cchat.SendableMessage
	cchat.Noncer
	cchat.Attachments

	// These methods are reserved for internal use.

	Author() text.Rich
	AuthorID() string
	AuthorAvatarURL() string // may be empty
	Files() []attachment.File
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

func (s SendMessageData) AsNoncer() cchat.Noncer {
	return s
}

func (s SendMessageData) Nonce() string {
	return s.nonce
}

func (s SendMessageData) Files() []attachment.File {
	return s.files
}

func (s SendMessageData) AsAttachments() cchat.Attachments {
	return s
}

func (s SendMessageData) Attachments() []cchat.MessageAttachment {
	var attachments = make([]cchat.MessageAttachment, len(s.files))
	for i, file := range s.files {
		attachments[i] = file.AsAttachment()
	}
	return attachments
}
