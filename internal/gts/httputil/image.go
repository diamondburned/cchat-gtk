package httputil

import (
	"context"
	"io"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type ImageContainer interface {
	SetFromPixbuf(*gdk.Pixbuf)
	SetFromAnimation(*gdk.PixbufAnimation)
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)

	// for internal use
	pbgetter
}

type ImageContainerSizer interface {
	ImageContainer
	SetSizeRequest(w, h int)
}

// AsyncImage loads an image. This method uses the cache.
func AsyncImage(img ImageContainer, url string, procs ...imgutil.Processor) {
	if url == "" {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	connectDestroyer(img, cancel)

	go syncImageFn(ctx, img, url, procs, func(l *gdk.PixbufLoader, gif bool) {
		l.Connect("area-prepared", areaPreparedFn(ctx, img, gif))
	})
}

// AsyncImageSized resizes using GdkPixbuf. This method does not use the cache.
func AsyncImageSized(img ImageContainerSizer, url string, w, h int, procs ...imgutil.Processor) {
	if url == "" {
		return
	}

	// Add a processor to resize.
	procs = imgutil.Prepend(imgutil.Resize(w, h), procs)

	ctx, cancel := context.WithCancel(context.Background())
	connectDestroyer(img, cancel)

	go syncImageFn(ctx, img, url, procs, func(l *gdk.PixbufLoader, gif bool) {
		l.Connect("size-prepared", func(l *gdk.PixbufLoader, imgW, imgH int) {
			w, h = imgutil.MaxSize(imgW, imgH, w, h)
			if w != imgW || h != imgH {
				l.SetSize(w, h)
				execIfCtx(ctx, func() { img.SetSizeRequest(w, h) })
			}
		})

		l.Connect("area-prepared", areaPreparedFn(ctx, img, gif))
	})
}

type pbgetter interface {
	GetPixbuf() *gdk.Pixbuf
	GetAnimation() *gdk.PixbufAnimation
	GetStorageType() gtk.ImageType
}

var _ pbgetter = (*gtk.Image)(nil)

func connectDestroyer(img ImageContainer, cancel func()) {
	img.Connect("destroy", func(img ImageContainer) {
		cancel()
		img.SetFromPixbuf(nil)
	})
}

func areaPreparedFn(ctx context.Context, img ImageContainer, gif bool) func(l *gdk.PixbufLoader) {
	return func(l *gdk.PixbufLoader) {
		if !gif {
			p, err := l.GetPixbuf()
			if err != nil {
				log.Error(errors.Wrap(err, "Failed to get pixbuf"))
				return
			}
			execIfCtx(ctx, func() { img.SetFromPixbuf(p) })
		} else {
			p, err := l.GetAnimation()
			if err != nil {
				log.Error(errors.Wrap(err, "Failed to get animation"))
				return
			}
			execIfCtx(ctx, func() { img.SetFromAnimation(p) })
		}
	}
}

func execIfCtx(ctx context.Context, fn func()) {
	gts.ExecAsync(func() {
		if ctx.Err() == nil {
			fn()
		}
	})
}

func syncImageFn(
	ctx context.Context,
	img ImageContainer,
	url string,
	procs []imgutil.Processor,
	middle func(l *gdk.PixbufLoader, gif bool),
) {

	r, err := get(ctx, url, true)
	if err != nil {
		log.Error(err)
		return
	}
	defer r.Body.Close()

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to make pixbuf loader"))
		return
	}

	gif := strings.Contains(url, ".gif")

	// This is a very important signal, so we must do it synchronously. Gotk3's
	// callback implementation requires all connects to be synchronous to a
	// certain thread.
	<-gts.ExecSync(func() {
		middle(l, gif)
	})

	// If we have processors, then write directly in there.
	if len(procs) > 0 {
		if !gif {
			err = imgutil.ProcessStream(l, r.Body, procs)
		} else {
			err = imgutil.ProcessAnimationStream(l, r.Body, procs)
		}
	} else {
		// Else, directly copy the body over.
		_, err = io.Copy(l, r.Body)
	}

	if err != nil {
		log.Error(errors.Wrap(err, "Error processing image"))
		return
	}

	if err := l.Close(); err != nil {
		log.Error(errors.Wrap(err, "Failed to close pixbuf"))
	}
}
