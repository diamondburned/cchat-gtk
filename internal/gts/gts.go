package gts

import (
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

// AfterFunc mimics time.AfterFunc's API but runs the callback inside the Gtk
// main loop.
func AfterFunc(d time.Duration, f func()) (stop func()) {
	h, err := glib.TimeoutAdd(
		uint(d.Milliseconds()),
		func() bool { f(); return true },
	)
	if err != nil {
		panic(err)
	}

	return func() { glib.SourceRemove(h) }
}

func EventIsRightClick(ev *gdk.Event) bool {
	keyev := gdk.EventButtonNewFromEvent(ev)
	return keyev.Type() == gdk.EVENT_BUTTON_PRESS && keyev.Button() == gdk.BUTTON_SECONDARY
}
