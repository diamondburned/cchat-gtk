package gts

import (
	"os"
	"sync"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var Args = append([]string{}, os.Args...)
var recvPool *sync.Pool

var App struct {
	*gtk.Application
	Window *gtk.ApplicationWindow
	Header *gtk.HeaderBar
}

func init() {
	gtk.Init(&Args)

	recvPool = &sync.Pool{
		New: func() interface{} {
			return make(chan struct{})
		},
	}

	App.Application, _ = gtk.ApplicationNew("com.github.diamondburned.gtkcord3", 0)
}

type Windower interface {
	Window() gtk.IWidget
}

type Headerer interface {
	Header() gtk.IWidget
}

// Above interfaces should be kept for modularity, but since this is an internal
// abstraction, we already know our application will implement both.
type WindowHeaderer interface {
	Windower
	Headerer
}

func Main(wfn func() WindowHeaderer) {
	App.Application.Connect("activate", func() {
		App.Header, _ = gtk.HeaderBarNew()
		App.Header.Show()
		App.Header.SetShowCloseButton(true)

		App.Window, _ = gtk.ApplicationWindowNew(App.Application)
		App.Window.Show()
		App.Window.SetTitlebar(App.Header)

		// Execute the function later, because we need it to run after
		// initialization.
		w := wfn()
		App.Window.Add(w.Window())
		App.Header.Add(w.Header())
	})

	// Use a special function to run the application. Exit with the appropriate
	// exit code.
	os.Exit(App.Run(Args))
}

// Async runs fn asynchronously, then runs the function it returns in the Gtk
// main thread.
func Async(fn func() (func(), error)) {
	go func() {
		f, err := fn()
		if err != nil {
			log.Error(err)
			return
		}

		glib.IdleAdd(f)
	}()
}

// ExecAsync executes function asynchronously in the Gtk main thread.
func ExecAsync(fn func()) {
	glib.IdleAdd(fn)
}

// ExecSync executes the function asynchronously, but returns a channel that
// indicates when the job is done.
func ExecSync(fn func()) <-chan struct{} {
	ch := recvPool.Get().(chan struct{})
	defer recvPool.Put(ch)

	glib.IdleAdd(func() {
		fn()
		close(ch)
	})

	return ch
}
