package httputil

import (
	"io"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// AsyncImage loads an image. This method uses the cache.
func AsyncImage(img *gtk.Image, url string, procs ...imgutil.Processor) {
	go asyncImage(img, url, procs...)
}

func asyncImage(img *gtk.Image, url string, procs ...imgutil.Processor) {
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
	gts.ExecSync(func() {
		l.Connect("area-prepared", func() {
			if gif {
				p, err := l.GetPixbuf()
				if err != nil {
					log.Error(errors.Wrap(err, "Failed to get pixbuf"))
					return
				}
				img.SetFromPixbuf(p)
			} else {
				p, err := l.GetAnimation()
				if err != nil {
					log.Error(errors.Wrap(err, "Failed to get animation"))
					return
				}
				img.SetFromAnimation(p)
			}
		})
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

// AsyncImageSized resizes using GdkPixbuf. This method does not use the cache.
func AsyncImageSized(img *gtk.Image, url string, w, h int, procs ...imgutil.Processor) {
	// TODO
	panic("TODO")
}
