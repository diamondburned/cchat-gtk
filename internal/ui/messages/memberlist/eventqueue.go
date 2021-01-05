package memberlist

import (
	"sync"

	"github.com/diamondburned/cchat-gtk/internal/gts"
)

type EventQueuer interface {
	Activate()
	Deactivate()
}

// eventQueue is a rough unbounded event queue. A zero-value instance is valid.
type eventQueue struct {
	mutex     sync.Mutex
	activated bool
	// idleQueue contains the incoming callbacks to update events. This is a
	// temporary hack to the issue of popovers disappearing when its parent
	// widget, that is the list box, changes. This slice should then contain all
	// those events to be executed only when the Popover is popped down.
	idleQueue []func()
}

func (evq *eventQueue) Add(fn func()) {
	evq.mutex.Lock()
	defer evq.mutex.Unlock()

	if evq.activated {
		evq.idleQueue = append(evq.idleQueue, fn)
	} else {
		gts.ExecLater(fn)
	}
}

func (evq *eventQueue) Activate() {
	evq.mutex.Lock()
	defer evq.mutex.Unlock()

	evq.activated = true
}

func (evq *eventQueue) pop() []func() {
	evq.mutex.Lock()
	defer evq.mutex.Unlock()

	popped := evq.idleQueue
	evq.idleQueue = nil
	evq.activated = false

	return popped
}

func (evq *eventQueue) Deactivate() {
	var popped = evq.pop()

	const chunkSz = 25

	// We shouldn't try and run more than a certain amount of callbacks within a
	// single loop, as it will freeze up the UI.
	for i := 0; i < len(popped); i += chunkSz {
		// Calculate the bounds in chunks.
		start, end := i, min(i+chunkSz, len(popped))

		gts.ExecLater(func() {
			for _, fn := range popped[start:end] {
				fn()
			}
		})
	}
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
