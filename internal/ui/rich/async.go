package rich

import (
	"context"
	"log"
	"reflect"
	"time"

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

// Reusable is the synchronization primitive to provide a method for
// asynchronous cancellation and reusability.
//
// It works by copying the ID (time) for each asynchronous operation. The
// operation then completes, and the ID is then compared again before being
// used. It provides a cancellation abstraction around the Gtk main thread.
//
// This struct is not thread-safe, as it relies on the Gtk main thread
// synchronization.
type Reusable struct {
	time   int64 // creation time, used as ID
	ctx    context.Context
	cancel func()

	swapfn   reflect.Value // reflect fn
	arg1type reflect.Type
}

var _ Reuser = (*Reusable)(nil)

func NewReusable(swapperFn interface{}) *Reusable {
	r := Reusable{}
	r.swapfn = reflect.ValueOf(swapperFn)
	r.arg1type = r.swapfn.Type().In(0)
	r.Invalidate()
	return &r
}

// Invalidate generates a new ID for the primitive, which would render
// asynchronously updating elements invalid.
func (r *Reusable) Invalidate() {
	// Cancel the old context.
	if r.cancel != nil {
		r.cancel()
	}

	// Reset.
	r.time = time.Now().UnixNano()
	r.ctx, r.cancel = context.WithCancel(context.Background())
}

// Context returns the reusable's cancellable context. It never returns nil.
func (r *Reusable) Context() context.Context {
	return r.ctx
}

// Reusable checks the acquired ID against the current one.
func (r *Reusable) Validate(acquired int64) (valid bool) {
	return r.time == acquired
}

// Acquire lends the ID to be given to Reusable() after finishing.
func (r *Reusable) Acquire() int64 {
	return r.time
}

func (r *Reusable) SwapResource(v interface{}, cancel func()) {
	r.swapfn.Call([]reflect.Value{reflect.ValueOf(v)})
}
