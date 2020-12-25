package httputil

import (
	"context"
	"io"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// TODO:

type ImageContainer interface {
	primitives.Connector

	SetFromPixbuf(*gdk.Pixbuf)
	SetFromAnimation(*gdk.PixbufAnimation)
	GetSizeRequest() (w, h int)
}

type SurfaceContainer interface {
	ImageContainer
	GetScaleFactor() int
	SetFromSurface(*cairo.Surface)
}

var (
	_ ImageContainer   = (*gtk.Image)(nil)
	_ SurfaceContainer = (*gtk.Image)(nil)
)

type surfaceWrapper struct {
	SurfaceContainer
	scale int
}

func (wrapper surfaceWrapper) SetFromPixbuf(pb *gdk.Pixbuf) {
	surface, _ := gdk.CairoSurfaceCreateFromPixbuf(pb, wrapper.scale, nil)
	wrapper.SetFromSurface(surface)
}

// AsyncImage loads an image. This method uses the cache. It prefers loading
// SetFromSurface over SetFromPixbuf, but will fallback if needed be.
func AsyncImage(ctx context.Context,
	img ImageContainer, url string, procs ...imgutil.Processor) {

	if url == "" {
		return
	}

	gif := strings.Contains(url, ".gif")
	scale := 1

	surfaceContainer, canSurface := img.(SurfaceContainer)

	if canSurface = canSurface && !gif; canSurface {
		// Only bother with this API if we even have HiDPI.
		if scale = surfaceContainer.GetScaleFactor(); scale > 1 {
			img = surfaceWrapper{surfaceContainer, scale}
		}
	}

	ctx = primitives.HandleDestroyCtx(ctx, img)

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to make pixbuf loader"))
		return
	}

	w, h := img.GetSizeRequest()
	l.Connect("size-prepared", func(l *gdk.PixbufLoader, imgW, imgH int) {
		w, h = imgutil.MaxSize(imgW, imgH, w, h)
		if w != imgW || h != imgH || scale > 1 {
			l.SetSize(w*scale, h*scale)
		}
	})

	l.Connect("area-prepared", areaPreparedFn(ctx, img, gif))

	go downloadImage(ctx, l, url, procs, gif)
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

func downloadImage(ctx context.Context, dst io.WriteCloser, url string, p []imgutil.Processor, gif bool) {
	// Close at the end when done.
	defer dst.Close()

	r, err := get(ctx, url, true)
	if err != nil {
		log.Error(err)
		return
	}
	defer r.Body.Close()

	// If we have processors, then write directly in there.
	if len(p) > 0 {
		if !gif {
			err = imgutil.ProcessStream(dst, r.Body, p)
		} else {
			err = imgutil.ProcessAnimationStream(dst, r.Body, p)
		}
	} else {
		// Else, directly copy the body over.
		_, err = io.Copy(dst, r.Body)
	}

	if err != nil {
		log.Error(errors.Wrap(err, "Error processing image"))
		return
	}
}
