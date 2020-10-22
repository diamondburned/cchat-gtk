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
		gts.ExecAsync(fn)
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

	// We shouldn't try and run more than a certain amount of callbacks within a
	// single loop, as it will freeze up the UI.
	if len(popped) > 25 {
		for _, fn := range popped {
			gts.ExecAsync(fn)
		}
		return
	}

	gts.ExecAsync(func() {
		for _, fn := range popped {
			fn()
		}
	})
}
