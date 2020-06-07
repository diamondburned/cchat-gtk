package httputil

import (
	"io"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/pkg/errors"
)

type ImageContainer interface {
	SetFromPixbuf(*gdk.Pixbuf)
	SetFromAnimation(*gdk.PixbufAnimation)
}

type ImageContainerSizer interface {
	ImageContainer
	SetSizeRequest(w, h int)
}

// AsyncImage loads an image. This method uses the cache.
func AsyncImage(img ImageContainer, url string, procs ...imgutil.Processor) {
	go syncImageFn(img, url, procs, func(l *gdk.PixbufLoader, gif bool) {
		l.Connect("area-prepared", areaPreparedFn(img, gif))
	})
}

// AsyncImageSized resizes using GdkPixbuf. This method does not use the cache.
func AsyncImageSized(img ImageContainerSizer, url string, w, h int, procs ...imgutil.Processor) {
	go syncImageFn(img, url, procs, func(l *gdk.PixbufLoader, gif bool) {
		l.Connect("size-prepared", func(l *gdk.PixbufLoader, imgW, imgH int) {
			w, h = imgutil.MaxSize(imgW, imgH, w, h)
			if w != imgW || h != imgH {
				l.SetSize(w, h)
				img.SetSizeRequest(w, h)
			}
		})

		l.Connect("area-prepared", areaPreparedFn(img, gif))
	})
}

func areaPreparedFn(img ImageContainer, gif bool) func(l *gdk.PixbufLoader) {
	return func(l *gdk.PixbufLoader) {
		if !gif {
			p, err := l.GetPixbuf()
			if err != nil {
				log.Error(errors.Wrap(err, "Failed to get pixbuf"))
				return
			}
			gts.ExecAsync(func() { img.SetFromPixbuf(p) })
		} else {
			p, err := l.GetAnimation()
			if err != nil {
				log.Error(errors.Wrap(err, "Failed to get animation"))
				return
			}
			gts.ExecAsync(func() { img.SetFromAnimation(p) })
		}
	}
}

func syncImageFn(
	img ImageContainer,
	url string,
	procs []imgutil.Processor,
	middle func(l *gdk.PixbufLoader, gif bool),
) {

	r, err := get(url, true)
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
