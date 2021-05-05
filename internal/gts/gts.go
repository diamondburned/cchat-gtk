package gts

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts/throttler"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const AppID = "com.github.diamondburned.cchat-gtk"

var Args = append([]string{}, os.Args...)

var App struct {
	*gtk.Application
	Window    *handy.ApplicationWindow
	Throttler *throttler.State

	closing bool
}

// IsClosing returns true if the window is destroyed.
func IsClosing() bool {
	return App.closing
}

// Windower is the interface for a window.
type Windower interface {
	gtk.IWidget
	gtk.IWindow
	throttler.Connector
}

func AddWindow(w Windower) {
	App.AddWindow(w)
	App.Throttler.Connect(w)
}

// Clipboard is initialized on init().
var Clipboard *gtk.Clipboard

// NewModalDialog returns a new modal dialog that's transient for the main
// window.
func NewModalDialog() (*gtk.Dialog, error) {
	d, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	d.SetModal(true)
	d.SetTransientFor(App.Window)

	AddWindow(d)

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
	b.Destroy()

	return d, nil
}

func AddAppAction(name string, call func()) {
	action := glib.SimpleActionNew(name, nil)
	action.Connect("activate", func(interface{}) { call() })
	App.AddAction(action)
}

func init() {
	gtk.Init(&Args)
	App.Application, _ = gtk.ApplicationNew(AppID, 0)
	Clipboard, _ = gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)

	// Limit the TPS of the main loop on window unfocus.
	App.Throttler = throttler.Bind(App.Application)
}

type MainApplication interface {
	gtk.IWidget
	Menu() *glib.MenuModel
	Icon() *gdk.Pixbuf // assume scale 1
	Close()
}

func Main(wfn func() MainApplication) {
	App.Application.Connect("activate", func(*gtk.Application) {
		handy.Init()

		// Load all CSS onto the default screen.
		loadProviders(getDefaultScreen())

		// App.Header, _ = gtk.HeaderBarNew()
		// // Right buttons only.
		// App.Header.SetDecorationLayout(":minimize,close")
		// App.Header.SetShowCloseButton(true)
		// App.Header.SetProperty("spacing", 0)

		App.Window = handy.ApplicationWindowNew()
		App.Window.SetDefaultSize(1000, 500)
		App.Window.Show()
		AddWindow(&App.Window.Window)

		App.Throttler.Connect(&App.Window.Window)

		// Execute the function later, because we need it to run after
		// initialization.
		w := wfn()
		App.Window.Add(w)
		App.Window.SetIcon(w.Icon())

		// Connect the destructor.
		App.Window.Window.Connect("destroy", func(window *handy.ApplicationWindow) {
			// Hide the application window.
			window.Hide()
			App.closing = true

			// Let the main loop run once by queueing the stop loop afterwards.
			// This is to allow the main loop to properly hide the Gtk window
			// before trying to disconnect.
			ExecLater(func() {
				// Stop the application loop.
				App.Application.Quit()
				// Finalize the application by running the closer.
				w.Close()
			})
		})

		// Connect extra actions.
		AddAppAction("quit", App.Window.Destroy)
	})

	// Use a special function to run the application. Exit with the appropriate
	// exit code if necessary.
	if code := App.Run(Args); code > 0 {
		os.Exit(code)
	}
}

// Async runs fn asynchronously, then runs the function it returns in the Gtk
// main thread.
// TODO: deprecate Async.
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

// AsyncCancel is similar to AsyncCtx, but the context is created internally.
func AsyncCancel(fn func(ctx context.Context) (func(), error)) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// fn() is assumed to use the same given ctx.
		f, err := fn(ctx)
		if err != nil {
			log.Error(err)
		}

		// Attempt to run the callback if it's there.
		if f != nil {
			ExecAsyncCtx(ctx, f)
		}
	}()

	return cancel
}

// AsyncCtx does what Async does, except the returned callback will not be
// executed if the given context has expired or the returned callback is called.
func AsyncCtx(ctx context.Context, fn func() (func(), error)) {
	go func() {
		// fn() is assumed to use the same given ctx.
		f, err := fn()
		if err != nil {
			log.Error(err)
		}

		// Attempt to run the callback if it's there.
		if f != nil {
			ExecAsyncCtx(ctx, f)
		}
	}()
}

// ExecLater executes the function asynchronously with a low priority.
func ExecLater(fn func()) {
	glib.IdleAddPriority(glib.PRIORITY_DEFAULT_IDLE, fn)
}

// ExecAsync executes function asynchronously in the Gtk main thread.
// TODO: deprecate Async.
func ExecAsync(fn func()) {
	glib.IdleAddPriority(glib.PRIORITY_HIGH, fn)
}

// ExecAsyncCtx executes the function asynchronously in the Gtk main thread only
// if the context has not expired. This API has absolutely no race conditions if
// the context is only canceled in the main thread.
func ExecAsyncCtx(ctx context.Context, fn func()) {
	ExecAsync(func() {
		select {
		case <-ctx.Done():

		default:
			fn()
		}
	})
}

// DoAfter calls f after the given duration in the Gtk main loop.
func DoAfter(d time.Duration, f func()) {
	DoAfterMs(uint(d.Milliseconds()), f)
}

// DoAfterMs calls f after the given ms in the Gtk main loop.
func DoAfterMs(ms uint, f func()) {
	if secs := ms / 1000; secs*1000 == ms {
		glib.TimeoutSecondsAddPriority(secs, glib.PRIORITY_HIGH_IDLE, f)
	} else {
		glib.TimeoutAddPriority(ms, glib.PRIORITY_HIGH_IDLE, f)
	}
}

// AfterFunc mimics time.AfterFunc's API but runs the callback inside the Gtk
// main loop.
func AfterFunc(d time.Duration, f func()) (stop func()) {
	return AfterMsFunc(uint(d.Milliseconds()), f)
}

// AfterMsFunc is similar to AfterFunc but takes in milliseconds instead.
func AfterMsFunc(ms uint, f func()) (stop func()) {
	fn := func() bool { f(); return true }

	var h glib.SourceHandle
	if secs := ms / 1000; secs*1000 == ms {
		h = glib.TimeoutSecondsAddPriority(secs, glib.PRIORITY_HIGH_IDLE, fn)
	} else {
		h = glib.TimeoutAddPriority(ms, glib.PRIORITY_HIGH_IDLE, fn)
	}

	return func() { glib.SourceRemove(h) }
}

func EventIsRightClick(ev *gdk.Event) bool {
	keyev := gdk.EventButtonNewFromEvent(ev)
	return keyev.Type() == gdk.EVENT_BUTTON_PRESS && keyev.Button() == gdk.BUTTON_SECONDARY
}

func EventIsLeftClick(ev *gdk.Event) bool {
	keyev := gdk.EventButtonNewFromEvent(ev)
	return keyev.Type() == gdk.EVENT_BUTTON_PRESS && keyev.Button() == gdk.BUTTON_PRIMARY
}

func SpawnUploader(dirpath string, callback func(absolutePaths []string)) {
	dialog, _ := gtk.FileChooserNativeDialogNew(
		"Upload File", App.Window,
		gtk.FILE_CHOOSER_ACTION_OPEN,
		"Upload", "Cancel",
	)

	// BindPreviewer(dialog)

	if dirpath == "" {
		p, err := os.Getwd()
		if err != nil {
			p = glib.GetUserDataDir()
		}
		dirpath = p
	}

	dialog.SetLocalOnly(false)
	dialog.SetCurrentFolder(dirpath)
	dialog.SetSelectMultiple(true)

	res := dialog.Run()
	dialog.Destroy()

	if res != int(gtk.RESPONSE_ACCEPT) {
		return
	}

	names, _ := dialog.GetFilenames()
	callback(names)
}

// BindPreviewer binds the file chooser dialog with a previewer.
func BindPreviewer(fc *gtk.FileChooserNativeDialog) {
	img, _ := gtk.ImageNew()

	fc.SetPreviewWidget(img)
	fc.Connect("update-preview", func(interface{}) { loadImage(fc, img) })
}

func loadImage(fc *gtk.FileChooserNativeDialog, img *gtk.Image) {
	file := fc.GetPreviewFilename()

	go func() {
		var animation *gdk.PixbufAnimation
		var pixbuf *gdk.Pixbuf

		defer ExecAsync(func() {
			if fc.GetPreviewFilename() == file {
				if animation == nil && pixbuf == nil {
					fc.SetPreviewWidgetActive(false)
					return
				}

				if animation != nil {
					img.SetFromAnimation(animation)
				} else {
					img.SetFromPixbuf(pixbuf)
				}

				fc.SetPreviewWidgetActive(true)
			}
		})

		l, err := gdk.PixbufLoaderNew()
		if err != nil {
			return
		}

		f, err := os.Open(file)
		if err != nil {
			return
		}
		defer f.Close()

		if _, err := io.Copy(l, f); err != nil {
			return
		}

		if err := l.Close(); err != nil {
			return
		}

		if pixbuf == nil {
			return
		}
	}()
}
