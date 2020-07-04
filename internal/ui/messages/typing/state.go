package typing

import (
	"sort"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/pkg/errors"
)

type State struct {
	// states
	typers      []cchat.Typer
	timeout     time.Duration
	canceler    func()
	invalidated bool

	// consts
	changed func(s *State, empty bool)
	stopper func() // stops the event loop, not used atm
}

var _ cchat.TypingIndicator = (*State)(nil)

func NewState(changed func(s *State, empty bool)) *State {
	s := &State{changed: changed}
	s.stopper = gts.AfterFunc(time.Second/2, s.loop)
	return s
}

func (s *State) reset() {
	if s.canceler != nil {
		s.canceler()
		s.canceler = nil
	}

	s.timeout = 0
	s.typers = nil
	s.invalidated = false
}

// Subscribe is thread-safe.
func (s *State) Subscribe(indicator cchat.ServerMessageTypingIndicator) {
	gts.Async(func() (func(), error) {
		c, err := indicator.TypingSubscribe(s)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to subscribe to typing indicator")
		}

		return func() {
			s.canceler = c
			s.timeout = indicator.TypingTimeout()
		}, nil
	})
}

// loop runs a single iteration of the event loop. This function is not
// thread-safe.
func (s *State) loop() {
	// Filter out any expired typers.
	t, ok := filterTypers(s.typers, s.timeout)
	if ok {
		s.invalidated = true
		s.typers = t
	}

	// Call the event handler if things are invalidated.
	if s.invalidated {
		s.changed(s, len(s.typers) == 0)
		s.invalidated = false
	}
}

// invalidate sorts and invalidates the state.
func (s *State) invalidate() {
	// Sort the list of typers again.
	sort.Slice(s.typers, func(i, j int) bool {
		return s.typers[i].Time().Before(s.typers[j].Time())
	})

	s.invalidated = true
}

// AddTyper is thread-safe.
func (s *State) AddTyper(typer cchat.Typer) {
	gts.ExecAsync(func() {
		defer s.invalidate()

		// If the typer already exists, then pop them to the start of the list.
		for i, t := range s.typers {
			if t.ID() == typer.ID() {
				s.typers[i] = t
				return
			}
		}

		s.typers = append(s.typers, typer)
	})
}

// RemoveTyper is thread-safe.
func (s *State) RemoveTyper(typerID string) {
	gts.ExecAsync(func() { s.removeTyper(typerID) })
}

func (s *State) removeTyper(typerID string) {
	defer s.invalidate()

	for i, t := range s.typers {
		if t.ID() == typerID {
			// Remove the quick way. Sort will take care of ordering.
			l := len(s.typers) - 1
			s.typers[i] = s.typers[l]
			s.typers[l] = nil
			s.typers = s.typers[:l]

			return
		}
	}
}

func filterTypers(typers []cchat.Typer, timeout time.Duration) ([]cchat.Typer, bool) {
	// Fast path.
	if len(typers) == 0 || timeout == 0 {
		return nil, false
	}

	var now = time.Now()
	var cut int

	for _, t := range typers {
		if now.Sub(t.Time()) < timeout {
			typers[cut] = t
			cut++
		}
	}

	for i := cut; i < len(typers); i++ {
		typers[i] = nil
	}

	var changed = cut != len(typers)
	return typers[:cut], changed
}
