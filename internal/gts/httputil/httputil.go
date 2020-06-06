package httputil

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/die-net/lrucache"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"github.com/pkg/errors"
)

var dskcached *http.Client
var memcached *http.Client

func init() {
	var basePath = filepath.Join(os.TempDir(), "cchat-gtk-pridemonth")

	http.DefaultClient.Timeout = 15 * time.Second

	dskcached = &(*http.DefaultClient)
	dskcached.Transport = httpcache.NewTransport(
		diskcache.NewWithDiskv(diskv.New(diskv.Options{
			BasePath:     basePath,
			TempDir:      filepath.Join(basePath, "tmp"),
			PathPerm:     0750,
			FilePerm:     0750,
			Compression:  diskv.NewZlibCompressionLevel(2),
			CacheSizeMax: 25 * 1024 * 1024, // 25 MiB in memory
		})),
	)

	memcached = &(*http.DefaultClient)
	memcached.Transport = httpcache.NewTransport(lrucache.New(
		25*1024*1024,      // 25 MiB in memory
		secs(2*time.Hour), // 2 hours cache
	))
}

func secs(dura time.Duration) int64 {
	return int64(dura / time.Second)
}

func AsyncStreamUncached(url string, fn func(r io.Reader)) {
	gts.Async(func() (func(), error) {
		r, err := get(url, false)
		if err != nil {
			return nil, err
		}

		return func() {
			fn(r.Body)
			r.Body.Close()
		}, nil
	})
}

func AsyncStream(url string, fn func(r io.Reader)) {
	gts.Async(func() (func(), error) {
		r, err := get(url, true)
		if err != nil {
			return nil, err
		}

		return func() {
			fn(r.Body)
			r.Body.Close()
		}, nil
	})
}

func get(url string, cached bool) (r *http.Response, err error) {
	if cached {
		r, err = dskcached.Get(url)
	} else {
		r, err = memcached.Get(url)
	}

	if err != nil {
		return nil, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		r.Body.Close()
		return nil, errors.Errorf("Unexpected status %d", r.StatusCode)
	}

	return r, nil
}
