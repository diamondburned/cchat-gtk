// +build prof

package main

import (
	_ "net/http/pprof"

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

import "C"

const ProfileAddr = "localhost:49583"

func init() {
	C.HeapProfilerStart()
	destructor = func() { C.HeapProfilerStop() }

	// runtime.SetBlockProfileRate(1)

	// go func() {
	// 	log.Println("Listening to profiler at", ProfileAddr)

	// 	if err := http.ListenAndServe(ProfileAddr, nil); err != nil {
	// 		log.Error(errors.Wrap(err, "Failed to start profiling HTTP server"))
	// 	}
	// }()

	// f, _ := os.Create("/tmp/cchat.pprof")
	// p := pprof.Lookup("block")

	// destructor = func() {
	// 	log.Println("==destructor==")

	// 	if err := p.WriteTo(f, 2); err != nil {
	// 		log.Println("Profile writeTo error:", err)
	// 	}

	// 	f.Close()
	// }

	// f, _ := os.Create("/tmp/cchat.pprof")
	// if err := pprof.StartCPUProfile(f); err != nil {
	// 	panic(err)
	// }

	// destructor = func() {
	// 	pprof.StopCPUProfile()
	// 	f.Close()
	// }
}
