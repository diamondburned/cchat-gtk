package keyring

import (
	"bytes"
	"encoding/gob"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/log"
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

func SaveSessions(serviceName string, sessions []Session) {
	if err := set(serviceName, sessions); err != nil {
		log.Warn(errors.Wrap(err, "Error saving session"))
	}
}

// RestoreSessions restores all sessions of the service asynchronously, then
// calls the auth callback inside the Gtk main thread.
func RestoreSessions(serviceName string) (sessions []Session) {
	// Ignore the error, it's not important.
	if err := get(serviceName, &sessions); err != nil {
		log.Warn(err)
	}
	return
}
