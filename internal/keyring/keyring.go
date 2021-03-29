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
	ID cchat.ID

	// Metadata.
	Name string
	Data map[string]string
}

// ConvertSession attempts to get the session data from the given cchat session.
// It returns nil if it can't do it.
func ConvertSession(ses cchat.Session, name string) *Session {
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

// RestoreSession restores a single session.
func RestoreSession(svc cchat.Service, sessionID cchat.ID) *Session {
	service := Restore(svc)
	for _, session := range service.Sessions {
		if session.ID == sessionID {
			return &session
		}
	}
	return nil
}

// Sessions is a list of sessions within a keyring. It provides an abstract way
// to save sessions with order.
type Service struct {
	ID       cchat.ID
	Sessions []Session
}

// NewService creates a new service.
func NewService(svc cchat.Service, cap int) Service {
	return Service{
		ID:       svc.ID(),
		Sessions: make([]Session, 0, cap),
	}
}

// Restore restores all sessions of the service asynchronously, then calls the
// auth callback inside the GTK main thread.
func Restore(svc cchat.Service) Service {
	var sessions []Session
	// Ignore the error, it's not important.
	if err := store.Get(svc.ID(), &sessions); err != nil {
		log.Warn(err)
	}

	return Service{
		ID:       svc.ID(),
		Sessions: sessions,
	}
}

// Add adds a session into the sessions list.
func (svc *Service) Add(ses cchat.Session, name string) {
	s := ConvertSession(ses, name)
	if s == nil {
		return
	}

	svc.Sessions = append(svc.Sessions, *s)
}

// Save saves the sessions into the keyring.
func (svc Service) Save() {
	if err := store.Set(svc.ID, svc.Sessions); err != nil {
		log.Warn(errors.Wrap(err, "Error saving session"))
	}
}
