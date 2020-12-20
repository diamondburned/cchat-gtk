package rich

import (
	"context"
	"log"

	"github.com/diamondburned/cchat-gtk/internal/gts"
)

// Reuser is an interface for structs that inherit Reusable.
type Reuser interface {
	Context() context.Context
	Acquire() int64
	Validate(int64) bool
	SwapResource(v interface{}, cancel func())
}

type AsyncUser = func(context.Context) (interface{}, func(), error)

// AsyncUse is a handler for structs that implement the Reuser primitive. The
// passed in function will be called asynchronously, but swap will be called in
// the Gtk main thread.
func AsyncUse(r Reuser, fn AsyncUser) {
	// Acquire an ID.
	id := r.Acquire()
	ctx := r.Context()

	gts.Async(func() (func(), error) {
		// Run the callback asynchronously.
		v, cancel, err := fn(ctx)
		if err != nil {
			return nil, err
		}

		return func() {
			// Validate the ID. Cancel if it's invalid.
			if !r.Validate(id) {
				log.Println("Async function value dropped for reusable primitive.")
				return
			}

			// Update the resource.
			r.SwapResource(v, cancel)
		}, nil
	})
}
