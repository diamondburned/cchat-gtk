// +build prof

package main

import (
	"os"
	"runtime/pprof"

	_ "net/http/pprof"

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

func init() {
	// go func() {
	// 	if err := http.ListenAndServe("localhost:42069", nil); err != nil {
	// 		log.Error(errors.Wrap(err, "Failed to start profiling HTTP server"))
	// 	}
	// }()

	// runtime.SetBlockProfileRate(1)

	// f, _ := os.Create("/tmp/cchat.pprof")
	// p := pprof.Lookup("block")

	// destructor = func() {
	// 	log.Println("==destructor==")

	// 	if err := p.WriteTo(f, 2); err != nil {
	// 		log.Println("Profile writeTo error:", err)
	// 	}

	// 	f.Close()
	// }

	f, _ := os.Create("/tmp/cchat.pprof")
	if err := pprof.StartCPUProfile(f); err != nil {
		panic(err)
	}

	destructor = func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}
