package httputil

import (
	"bufio"
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// bufferPool provides a sync.Pool of *bufio.Writers. This is used to reduce the
// amount of cgo calls, by writing bytes in larger chunks.
//
// Technically, httpcache already wraps its cached reader around a bufio.Reader,
// but we have no control over the buffer size.
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Allocate a 512KB buffer by default.
		const defaultBufSz = 512 * 1024

		return bufio.NewWriterSize(nil, defaultBufSz)
	},
}

func bufferedWriter(w io.Writer) *bufio.Writer {
	buf := bufferPool.Get().(*bufio.Writer)
	buf.Reset(w)
	return buf
}

func returnBufferedWriter(buf *bufio.Writer) {
	// Unreference the internal reader.
	buf.Reset(nil)
	bufferPool.Put(buf)
}

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

func (wrapper *surfaceWrapper) SetFromPixbuf(pb *gdk.Pixbuf) {
	surface, _ := gdk.CairoSurfaceCreateFromPixbuf(pb, wrapper.scale, nil)
	wrapper.SetFromSurface(surface)
}

// AsyncImage loads an image. This method uses the cache. It prefers loading
// SetFromSurface over SetFromPixbuf, but will fallback if needed be.
func AsyncImage(ctx context.Context,
	img ImageContainer, imageURL string, procs ...imgutil.Processor) {

	if imageURL == "" {
		return
	}

	w, h := img.GetSizeRequest()
	scale := 1

	surfaceContainer, canSurface := img.(SurfaceContainer)
	if canSurface {
		scale = surfaceContainer.GetScaleFactor()
	}

	ctx = primitives.HandleDestroyCtx(ctx, img)

	go func() {
		// Try and guess the MIME type from the URL.
		mimeType := mime.TypeByExtension(urlExt(imageURL))

		r, err := get(ctx, imageURL, true)
		if err != nil {
			log.Error(errors.Wrap(err, "failed to GET"))
			return
		}
		defer r.Body.Close()

		// Try and use the image type from the MIME header over the type from
		// the URL, as it is more reliable.
		if mime := mimeFromHeaders(r.Header); mime != "" {
			mimeType = mime
		}

		_, fileType := path.Split(mimeType) // abuse split "a/b" to get b

		isGIF := fileType == "gif"
		if isGIF {
			canSurface = false
			scale = 1
		}

		// Only bother with this if we even have HiDPI. We also can't use a
		// Surface for a GIF.
		if canSurface && scale > 1 {
			img = &surfaceWrapper{surfaceContainer, scale}
		}

		l, err := gdk.PixbufLoaderNewWithType(fileType)
		if err != nil {
			log.Error(errors.Wrapf(err, "failed to make PixbufLoader type %q", fileType))
			return
		}

		l.Connect("size-prepared", func(l *gdk.PixbufLoader, imgW, imgH int) {
			w, h = imgutil.MaxSize(imgW, imgH, w, h)
			if w != imgW || h != imgH || scale > 1 {
				l.SetSize(w*scale, h*scale)
			}
		})

		load := loadFn(ctx, img, isGIF)
		l.Connect("area-prepared", load)
		l.Connect("area-updated", load)

		// Borrow a buffered writer and return it at the end.
		bufWriter := bufferedWriter(l)
		defer returnBufferedWriter(bufWriter)

		if err := downloadImage(r.Body, bufWriter, procs, isGIF); err != nil {
			log.Error(errors.Wrapf(err, "failed to download %q", imageURL))
			// Force close after downloading.
		}

		if err := bufWriter.Flush(); err != nil {
			log.Error(errors.Wrapf(err, "failed to flush writer for %q", imageURL))
			// Force close after downloading.
		}

		if err := l.Close(); err != nil {
			log.Error(errors.Wrapf(err, "failed to close pixbuf loader for %q", imageURL))
		}
	}()
}

func urlExt(anyURL string) string {
	u, err := url.Parse(anyURL)
	if err != nil {
		return path.Ext(strings.SplitN(anyURL, "?", 1)[0])
	}

	return path.Ext(u.Path)
}

func mimeFromHeaders(headers http.Header) string {
	cType := headers.Get("Content-Type")
	if cType == "" {
		return ""
	}
	media, _, err := mime.ParseMediaType(cType)
	if err != nil {
		return ""
	}
	return media
}

func loadFn(ctx context.Context, img ImageContainer, isGIF bool) func(l *gdk.PixbufLoader) {
	var pixbuf interface{}

	return func(l *gdk.PixbufLoader) {
		if pixbuf == nil {
			if !isGIF {
				pixbuf, _ = l.GetPixbuf()
			} else {
				pixbuf, _ = l.GetAnimation()
			}
		}

		switch pixbuf := pixbuf.(type) {
		case *gdk.Pixbuf:
			execIfCtx(ctx, func() { img.SetFromPixbuf(pixbuf) })
		case *gdk.PixbufAnimation:
			execIfCtx(ctx, func() { img.SetFromAnimation(pixbuf) })
		}
	}
}

func execIfCtx(ctx context.Context, fn func()) {
	gts.ExecLater(func() {
		if ctx.Err() == nil {
			fn()
		}
	})
}

func downloadImage(src io.Reader, dst io.Writer, p []imgutil.Processor, isGIF bool) error {
	var err error

	// If we have processors, then write directly in there.
	if len(p) > 0 {
		if !isGIF {
			err = imgutil.ProcessStream(dst, src, p)
		} else {
			err = imgutil.ProcessAnimationStream(dst, src, p)
		}
	} else {
		// Else, directly copy the body over.
		_, err = io.Copy(dst, src)
	}

	if err != nil {
		return errors.Wrap(err, "failed to process image")
	}

	return nil
}
