package httputil

import (
	"context"
	"io"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/pkg/errors"
)

// TODO:

type ImageContainer interface {
	primitives.Connector

	SetFromPixbuf(*gdk.Pixbuf)
	SetFromAnimation(*gdk.PixbufAnimation)
	GetSizeRequest() (w, h int)
}

// AsyncImage loads an image. This method uses the cache.
func AsyncImage(ctx context.Context,
	img ImageContainer, url string, procs ...imgutil.Processor) {

	if url == "" {
		return
	}

	ctx = primitives.HandleDestroyCtx(ctx, img)

	gif := strings.Contains(url, ".gif")

	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to make pixbuf loader"))
		return
	}

	if w, h := img.GetSizeRequest(); w > 0 && h > 0 {
		l.Connect("size-prepared", func(l *gdk.PixbufLoader, imgW, imgH int) {
			w, h = imgutil.MaxSize(imgW, imgH, w, h)
			if w != imgW || h != imgH {
				l.SetSize(w, h)
			}
		})
	}

	l.Connect("area-prepared", areaPreparedFn(ctx, img, gif))

	go syncImage(ctx, l, url, procs, gif)
}

// func connectDestroyer(img ImageContainer, cancel func()) {
// 	img.Connect("destroy", func() {
// 		cancel()
// 		img.SetFromPixbuf(nil)
// 	})
// }

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

func syncImage(ctx context.Context, l io.WriteCloser, url string, p []imgutil.Processor, gif bool) {
	// Close at the end when done.
	defer l.Close()

	r, err := get(ctx, url, true)
	if err != nil {
		log.Error(err)
		return
	}
	defer r.Body.Close()

	// If we have processors, then write directly in there.
	if len(p) > 0 {
		if !gif {
			err = imgutil.ProcessStream(l, r.Body, p)
		} else {
			err = imgutil.ProcessAnimationStream(l, r.Body, p)
		}
	} else {
		// Else, directly copy the body over.
		_, err = io.Copy(l, r.Body)
	}

	if err != nil {
		log.Error(errors.Wrap(err, "Error processing image"))
		return
	}
}
