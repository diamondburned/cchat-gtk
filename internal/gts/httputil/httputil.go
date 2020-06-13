package httputil

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"github.com/pkg/errors"
)

var dskcached *http.Client

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
}

func secs(dura time.Duration) int64 {
	return int64(dura / time.Second)
}

func AsyncStreamUncached(url string, fn func(r io.Reader)) {
	gts.Async(func() (func(), error) {
		r, err := get(context.Background(), url, false)
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
		r, err := get(context.Background(), url, true)
		if err != nil {
			return nil, err
		}

		return func() {
			fn(r.Body)
			r.Body.Close()
		}, nil
	})
}

func get(ctx context.Context, url string, cached bool) (r *http.Response, err error) {
	// if cached {
	// 	r, err = dskcached.Get(url)
	// } else {
	// 	r, err = memcached.Get(url)
	// }

	q, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to make a request")
	}

	r, err = dskcached.Do(q)
	if err != nil {
		return nil, err
	}

	if r.StatusCode < 200 || r.StatusCode > 299 {
		r.Body.Close()
		return nil, errors.Errorf("Unexpected status %d", r.StatusCode)
	}

	return r, nil
}
