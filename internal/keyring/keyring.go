package keyring

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/keyring/driver"
	"github.com/diamondburned/cchat-gtk/internal/keyring/driver/json"
	"github.com/diamondburned/cchat-gtk/internal/keyring/driver/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

// Declare a keyring store with fallbacks.
var store = driver.NewStore(
	keyring.NewProvider(),
	json.NewProvider(config.DirPath()), // fallback
)

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
	if err := store.Set(serviceName.Content, sessions); err != nil {
		log.Warn(errors.Wrap(err, "Error saving session"))
	}
}

// RestoreSessions restores all sessions of the service asynchronously, then
// calls the auth callback inside the Gtk main thread.
func RestoreSessions(serviceName text.Rich) (sessions []Session) {
	// Ignore the error, it's not important.
	if err := store.Get(serviceName.Content, &sessions); err != nil {
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
