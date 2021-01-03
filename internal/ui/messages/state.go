package messages

import (
	"time"

	"github.com/diamondburned/cchat"
)

// ServerMessage combines Server and ServerMessage from cchat.
type ServerMessage interface {
	cchat.Server
	cchat.Messenger
}

type state struct {
	session cchat.Session
	server  cchat.Server

	actioner   cchat.Actioner
	backlogger cchat.Backlogger

	current func() // stop callback
	author  string

	lastBacklogged time.Time
}

func (s *state) Reset() {
	// If we still have the last server to leave, then leave it.
	if s.current != nil {
		s.current()
	}

	// Lazy way to reset the state.
	*s = state{}
}

func (s *state) hasActions() bool {
	return s.actioner != nil
}

// SessionID returns the session ID, or an empty string if there's no session.
func (s *state) SessionID() string {
	if s.session != nil {
		return s.session.ID()
	}
	return ""
}

// ServerID returns the server ID, or an empty string if there's no server.
func (s *state) ServerID() string {
	if s.server != nil {
		return s.server.ID()
	}
	return ""
}

const backloggingFreq = time.Second * 3

// Backlogger returns the backlogger instance if it's allowed to fetch more
// backlogs.
func (s *state) Backlogger() cchat.Backlogger {
	if s.backlogger == nil || s.current == nil {
		return nil
	}

	var now = time.Now()

	if s.lastBacklogged.Add(backloggingFreq).After(now) {
		return nil
	}

	s.lastBacklogged = now
	return s.backlogger
}

func (s *state) bind(session cchat.Session, server cchat.Server, msgr cchat.Messenger) {
	s.session = session
	s.server = server
	s.actioner = msgr.AsActioner()
	s.backlogger = msgr.AsBacklogger()
}

func (s *state) setcurrent(fn func()) {
	s.current = fn
}
