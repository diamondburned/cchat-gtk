package gts

import (
	"context"
	"os"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const AppID = "com.github.diamondburned.cchat-gtk"

var Args = append([]string{}, os.Args...)

var App struct {
	*gtk.Application
	Window *gtk.ApplicationWindow
	Header *gtk.HeaderBar
}

// NewModalDialog returns a new modal dialog that's transient for the main
// window.
func NewModalDialog() (*gtk.Dialog, error) {
	d, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	d.SetModal(true)
	d.SetTransientFor(App.Window)

	return d, nil
}

func NewEmptyModalDialog() (*gtk.Dialog, error) {
	d, err := NewModalDialog()
	if err != nil {
		return nil, err
	}

	b, err := d.GetContentArea()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get content area")
	}

	d.Remove(b)

	return d, nil
}

func AddAppAction(name string, call func()) {
	action := glib.SimpleActionNew(name, nil)
	action.Connect("activate", call)
	App.AddAction(action)
}

func AddWindowAction(name string, call func()) {
	action := glib.SimpleActionNew(name, nil)
	action.Connect("activate", call)
	App.Window.AddAction(action)
}

func init() {
	gtk.Init(&Args)
	App.Application, _ = gtk.ApplicationNew(AppID, 0)
}

type WindowHeaderer interface {
	Window() gtk.IWidget
	Header() gtk.IWidget
	Close()
}

func Main(wfn func() WindowHeaderer) {
	App.Application.Connect("activate", func() {
		// Load all CSS onto the default screen.
		loadProviders(getDefaultScreen())

		App.Header, _ = gtk.HeaderBarNew()
		App.Header.SetShowCloseButton(true)
		App.Header.Show()

		b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
		App.Header.SetCustomTitle(b)

		App.Window, _ = gtk.ApplicationWindowNew(App.Application)
		App.Window.SetDefaultSize(1000, 500)
		App.Window.SetTitlebar(App.Header)

		App.Window.Show()

		// Execute the function later, because we need it to run after
		// initialization.
		w := wfn()
		App.Window.Add(w.Window())
		App.Header.Add(w.Header())

		// Connect extra actions.
		AddAppAction("quit", App.Window.Destroy)

		// Connect the destructor.
		App.Window.Connect("destroy", func() {
			// Hide the application window.
			App.Window.Hide()

			// Let the main loop run once by queueing the stop loop afterwards.
			// This is to allow the main loop to properly hide the Gtk window
			// before trying to disconnect.
			ExecAsync(func() {
				// Stop the application loop.
				App.Application.Quit()
				// Finalize the application by running the closer.
				w.Close()
			})
		})
	})

	// Use a special function to run the application. Exit with the appropriate
	// exit code if necessary.
	if code := App.Run(Args); code > 0 {
		os.Exit(code)
	}
}

// Async runs fn asynchronously, then runs the function it returns in the Gtk
// main thread.
func Async(fn func() (func(), error)) {
	go func() {
		f, err := fn()
		if err != nil {
			log.Error(err)
		}

		// Attempt to run the callback if it's there.
		if f != nil {
			ExecAsync(f)
		}
	}()
}

// ExecAsync executes function asynchronously in the Gtk main thread.
func ExecAsync(fn func()) {
	glib.IdleAdd(fn)
}

// ExecSync executes the function asynchronously, but returns a channel that
// indicates when the job is done.
func ExecSync(fn func()) <-chan struct{} {
	var ch = make(chan struct{})

	glib.IdleAdd(func() {
		fn()
		close(ch)
	})

	return ch
}

func EventIsRightClick(ev *gdk.Event) bool {
	keyev := gdk.EventButtonNewFromEvent(ev)
	return keyev.Type() == gdk.EVENT_BUTTON_PRESS && keyev.Button() == gdk.BUTTON_SECONDARY
}

// Reuser is an interface for structs that inherit Reusable.
type Reuser interface {
	Context() context.Context
	Acquire() int64
	Validate(int64) bool
}

type AsyncUser = func(context.Context) (interface{}, error)

// AsyncUse is a handler for structs that implement the Reuser primitive. The
// passed in function will be called asynchronously, but swap will be called in
// the Gtk main thread.
func AsyncUse(r Reuser, swap func(interface{}), fn AsyncUser) {
	// Acquire an ID.
	id := r.Acquire()
	ctx := r.Context()

	Async(func() (func(), error) {
		// Run the callback asynchronously.
		v, err := fn(ctx)
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
			swap(v)
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
}

func NewReusable() *Reusable {
	r := &Reusable{}
	r.Invalidate()
	return r
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
