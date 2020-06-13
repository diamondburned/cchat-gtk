// +build prof

package main

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/pkg/errors"
)

func init() {
	go func() {
		if err := http.ListenAndServe("localhost:42069", nil); err != nil {
			log.Error(errors.Wrap(err, "Failed to start profiling HTTP server"))
		}
	}()
}
