package keyring

import (
	"bytes"
	"encoding/gob"
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/pkg/errors"
	"github.com/zalando/go-keyring"
)

func getThenDestroy(service string, v interface{}) error {
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

func SaveSessions(service cchat.Service, sessions []cchat.Session) (saveErrs []error) {
	var sessionData = make([]map[string]string, 0, len(sessions))

	for _, session := range sessions {
		sv, ok := session.(cchat.SessionSaver)
		if !ok {
			continue
		}

		d, err := sv.Save()
		if err != nil {
			saveErrs = append(saveErrs, err)
			continue
		}

		sessionData = append(sessionData, d)
	}

	if err := set(service.Name(), sessionData); err != nil {
		log.Warn(errors.Wrap(err, "Error saving session"))
		saveErrs = append(saveErrs, err)
	}

	return
}

// RestoreSessions restores all sessions of the service asynchronously, then
// calls the auth callback inside the Gtk main thread.
func RestoreSessions(service cchat.Service, auth func(cchat.Session)) {
	// If the service doesn't support restoring, treat it as a non-error.
	restorer, ok := service.(cchat.SessionRestorer)
	if !ok {
		return
	}

	var sessionData []map[string]string

	// Ignore the error, it's not important.
	if err := getThenDestroy(service.Name(), &sessionData); err != nil {
		log.Warn(err)
		return
	}

	for _, data := range sessionData {
		gts.Async(func() (func(), error) {
			s, err := restorer.RestoreSession(data)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to restore")
			}

			return func() { auth(s) }, nil
		})
	}
}
