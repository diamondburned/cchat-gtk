// Code generated by goprofiler. DO NOT EDIT.

package main

import (
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() {
		println("Serving HTTP at 127.0.0.1:48574 for profiler at /debug/pprof")
		panic(http.ListenAndServe("127.0.0.1:48574", nil))
	}()
}