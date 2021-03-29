// Package traverse implements an extensible interface that allows children
// widgets to announce state changes to their parent container.
//
// The objective of this package is to allow for easier parent traversal without
// cluttering its structure with too much state. It also allows for proper
// encapsulation as well as a fallback mechanism without lengthy boolean checks.
package traverse

import (
	"github.com/diamondburned/cchat"
)

// Breadcrumber is the base interface that other interfaces extend on. A child
// must at minimum implement this interface to use any other.
type Breadcrumber interface {
	// Breadcrumb returns the parent's path before the children's breadcrumb.
	// This method recursively joins the parent's crumb with the children's,
	// then eventually make its way up to the root node.
	ParentBreadcrumb() Breadcrumber
}

type BreadcrumbNamer interface {
	// Breadcrumb returns the breadcrumb name.
	Breadcrumb() string

	// TODO: make BreadcrumbNamer return LabelState.
}

// Traverse traverses the given breadcrumber recursively. If traverser returns
// true, then the function halts. Traversal is done from parent down to
// children.
func Traverse(bc Breadcrumber, traverser func(b Breadcrumber) bool) {
	if bc == nil {
		return
	}

	var stack []Breadcrumber
	for current := bc; current != nil; current = current.ParentBreadcrumb() {
		stack = append(stack, current)
	}

	for _, bc := range stack {
		if traverser(bc) {
			return
		}
	}
}

// TryBreadcrumb accepts a nilable breadcrumber and handles it appropriately.
func TryBreadcrumb(i Breadcrumber) (breadcrumbs []string) {
	Traverse(i, func(b Breadcrumber) bool {
		if namer, ok := b.(BreadcrumbNamer); ok {
			breadcrumbs = append(breadcrumbs, namer.Breadcrumb())
		}
		return false
	})

	for l, r := 0, len(breadcrumbs)-1; l < r; l, r = l+1, r-1 {
		breadcrumbs[l], breadcrumbs[r] = breadcrumbs[r], breadcrumbs[l]
	}

	return
}

func TryID(i Breadcrumber, appended ...cchat.ID) (ids []cchat.ID) {
	Traverse(i, func(b Breadcrumber) bool {
		switch b := b.(type) {
		case cchat.Identifier:
			ids = append(ids, b.ID())
		case BreadcrumbNamer:
			ids = append(ids, b.Breadcrumb())
		}

		return false
	})

	for l, r := 0, len(ids)-1; l < r; l, r = l+1, r-1 {
		ids[l], ids[r] = ids[r], ids[l]
	}

	return
}

// Unreadabler extends Breadcrumber to add unread states to the parent node.
type Unreadabler interface {
	SetState(id string, unread, mentioned bool)
}

// TrySetUnread tries to check if a breadcrumber parent node supports
// Unreadabler. If it does, then this function will set the state appropriately.
func TrySetUnread(parent Breadcrumber, selfID string, unread, mentioned bool) {
	if u, ok := parent.(Unreadabler); ok {
		u.SetState(selfID, unread, mentioned)
	}
}

// UnreadSetter is an interface that a single row implements to set state. It
// does not have to do with Breadcrumber.
type UnreadSetter interface {
	SetUnreadUnsafe(unread, mentioned bool)
}

// Unreadable is a struct that nodes could embed to implement unreadable
// capability, that is, the unread and mentioned states. A zero-value Unreadable
// is a valid Unreadable without an update handler.
//
// Typically, parent nodes would implement this as a way to count the number of
// unread and mentioned children nodes.
type Unreadable struct {
	UnreadableState
	unreadHandler func(unread, mentioned bool)
}

func NewUnreadable(unreadHandler UnreadSetter) *Unreadable {
	u := &Unreadable{}
	u.SetUnreadHandler(unreadHandler.SetUnreadUnsafe)
	return u
}

// SetUpdateHandler sets the parent's update handler. This update handler must
// refer to the parent's breadcrumb.
func (u *Unreadable) SetUnreadHandler(updateHandler func(unread, mentioned bool)) {
	// Update with the current state.
	if u.unreadHandler = updateHandler; updateHandler != nil {
		updateHandler(u.State())
	}
}

// SetState updates the node ID's state in this parent unreadable state
// container.
func (u *Unreadable) SetState(id string, unread, mentioned bool) {
	u.UnreadableState.SetState(id, unread, mentioned)

	if u.unreadHandler != nil {
		u.unreadHandler(u.UnreadableState.State())
	}
}

// UnreadableState implements a map of unread children for indication. A
// zero-value UnreadableState is a valid value.
type UnreadableState struct {
	// both maps represent sets of server IDs
	unreads  map[string]struct{}
	mentions map[string]struct{}
}

func NewUnreadableState() *UnreadableState {
	return &UnreadableState{}
}

func (s *UnreadableState) Reset() {
	s.unreads = map[string]struct{}{}
	s.mentions = map[string]struct{}{}
}

func (s *UnreadableState) State() (unread, mentioned bool) {
	unread = len(s.unreads) > 0
	mentioned = len(s.mentions) > 0

	// Count mentioned as unread.
	return unread || mentioned, mentioned
}

func (s *UnreadableState) SetState(id string, unread, mentioned bool) {
	if s.unreads == nil && s.mentions == nil {
		s.Reset()
	}

	setIf(unread, id, s.unreads)
	setIf(mentioned, id, s.mentions)
}

// setIf sets the ID into the given map if the cond boolean is true, or deletes
// it if the boolean is false.
func setIf(cond bool, id string, m map[string]struct{}) {
	if cond {
		m[id] = struct{}{}
	} else {
		delete(m, id)
	}
}
