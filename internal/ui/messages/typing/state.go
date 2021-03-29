package typing

import (
	"context"
	"sort"
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/pkg/errors"
)

type typer struct {
	cchat.User
	s *rich.NameContainer
	t time.Time
}

type State struct {
	// states
	typers      []typer
	timeout     time.Duration
	canceler    func()
	invalidated bool

	// consts
	changed func(s *State, empty bool)
	stopper func()
}

var _ cchat.TypingContainer = (*State)(nil)

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
func (s *State) Subscribe(indicator cchat.TypingIndicator) {
	gts.Async(func() (func(), error) {
		c, err := indicator.TypingSubscribe(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to subscribe to typing indicator")
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
		s.update()
		s.invalidated = false
	}
}

// update force-runs th callback.
func (s *State) update() {
	s.changed(s, len(s.typers) == 0)
}

// invalidate sorts and invalidates the state.
func (s *State) invalidate() {
	// Sort the list of typers again.
	sort.Slice(s.typers, func(i, j int) bool {
		return s.typers[i].t.Before(s.typers[j].t)
	})

	s.invalidated = true
}

// AddTyper is thread-safe.
func (s *State) AddTyper(user cchat.User) {
	now := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)

	state := rich.NameContainer{}
	state.QueueNamer(ctx, user)

	gts.ExecAsync(func() {
		defer cancel()
		defer s.invalidate()

		// If the typer already exists, then pop them to the start of the list.
		for i, t := range s.typers {
			if t.ID() == user.ID() {
				s.typers[i] = t
				return
			}
		}

		state.OnUpdate(s.update)

		s.typers = append(s.typers, typer{
			User: user,
			s:    &state,
			t:    now,
		})
	})
}

// RemoveTyper is thread-safe.
func (s *State) RemoveTyper(typerID string) {
	gts.ExecAsync(func() { s.removeTyper(typerID) })
}

func (s *State) removeTyper(typerID string) {
	defer s.invalidate()

	for i, t := range s.typers {
		if t.ID() != typerID {
			continue
		}

		// Invalidate the typer's label state.
		t.s.Stop()

		// Remove the quick way. Sort will take care of ordering.
		l := len(s.typers) - 1
		s.typers[i] = s.typers[l]
		s.typers[l] = typer{}
		s.typers = s.typers[:l]

		return
	}
}

func filterTypers(typers []typer, timeout time.Duration) ([]typer, bool) {
	// Fast path.
	if len(typers) == 0 || timeout == 0 {
		return nil, false
	}

	var now = time.Now()
	var cut int

	for _, t := range typers {
		if now.Sub(t.t) < timeout {
			typers[cut] = t
			cut++
		}
	}

	for i := cut; i < len(typers); i++ {
		typers[i].s.Stop()
		typers[i] = typer{}
	}

	var changed = cut != len(typers)
	return typers[:cut], changed
}
