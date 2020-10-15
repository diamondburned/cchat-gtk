package keyring

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/keyring/driver"
	"github.com/diamondburned/cchat-gtk/internal/keyring/driver/json"
	"github.com/diamondburned/cchat-gtk/internal/keyring/driver/keyring"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
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

// ConvertSession attempts to get the session data from the given cchat session.
// It returns nil if it can't do it.
func ConvertSession(ses cchat.Session) *Session {
	var name = ses.Name().Content

	saver := ses.AsSessionSaver()
	if saver == nil {
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
		Data: saver.SaveSession(),
	}
}

func SaveSessions(service cchat.Service, sessions []Session) {
	if err := store.Set(service.Name().Content, sessions); err != nil {
		log.Warn(errors.Wrap(err, "Error saving session"))
	}
}

// RestoreSessions restores all sessions of the service asynchronously, then
// calls the auth callback inside the Gtk main thread.
func RestoreSessions(service cchat.Service) (sessions []Session) {
	// Ignore the error, it's not important.
	if err := store.Get(service.Name().Content, &sessions); err != nil {
		log.Warn(err)
	}
	return
}

func RestoreSession(service cchat.Service, id string) *Session {
	var sessions = RestoreSessions(service)
	for _, session := range sessions {
		if session.ID == id {
			return &session
		}
	}
	return nil
}
