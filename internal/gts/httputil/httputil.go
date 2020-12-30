package httputil

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
	"github.com/pkg/errors"
)

var basePath = filepath.Join(os.TempDir(), "cchat-gtk-totally-not-node-modules")

var dskcached = http.Client{
	Timeout: 15 * time.Second,
	Transport: httpcache.NewTransport(
		diskcache.NewWithDiskv(diskv.New(diskv.Options{
			BasePath:     basePath,
			TempDir:      filepath.Join(basePath, "tmp"),
			PathPerm:     0750,
			FilePerm:     0750,
			Compression:  diskv.NewZlibCompressionLevel(5),
			CacheSizeMax: 0, // 25 MiB in memory
		})),
	),
}

func get(ctx context.Context, url string, cached bool) (r *http.Response, err error) {
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
