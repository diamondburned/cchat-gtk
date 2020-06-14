package keyring

import (
	"bytes"
	"encoding/gob"
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
	"github.com/zalando/go-keyring"
)

func get(service string, v interface{}) error {
	s, err := keyring.Get("cchat-gtk", service)
	if err != nil {
		return err
	}

	// Deleting immediately does not work on a successful start-up.
	// keyring.Delete("cchat-gtk", service)

	return gob.NewDecoder(strings.NewReader(s)).Decode(v)
}

func set(service string, v interface{}) error {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(v); err != nil {
		return err
	}

	return keyring.Set("cchat-gtk", service, b.String())
}

type Session struct {
	ID   string
	Name string
	Data map[string]string
}

func ConvertSession(ses cchat.Session, name string) *Session {
	saver, ok := ses.(cchat.SessionSaver)
	if !ok {
		return nil
	}

	s, err := saver.Save()
	if err != nil {
		log.Error(errors.Wrapf(err, "Failed to save session ID %s (%s)", ses.ID(), name))
		return nil
	}

	// Treat the ID as name if none is provided. This is a shitty hack around
	// backends that only set the name after returning.
	if name == "" {
		name = ses.ID()
	}

	return &Session{
		ID:   ses.ID(),
		Name: name,
		Data: s,
	}
}

func SaveSessions(serviceName text.Rich, sessions []Session) {
	if err := set(serviceName.Content, sessions); err != nil {
		log.Warn(errors.Wrap(err, "Error saving session"))
	}
}

// RestoreSessions restores all sessions of the service asynchronously, then
// calls the auth callback inside the Gtk main thread.
func RestoreSessions(serviceName text.Rich) (sessions []Session) {
	// Ignore the error, it's not important.
	if err := get(serviceName.Content, &sessions); err != nil {
		log.Warn(err)
	}
	return
}

func RestoreSession(serviceName text.Rich, id string) *Session {
	var sessions = RestoreSessions(serviceName)
	for _, session := range sessions {
		if session.ID == id {
			return &session
		}
	}
	return nil
}
