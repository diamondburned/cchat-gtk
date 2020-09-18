package main

import (
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat/services"

	_ "github.com/diamondburned/cchat-discord"
	_ "github.com/diamondburned/cchat-mock"
)

// destructor is used for debugging and profiling.
var destructor = func() {}

func init() {
	// Aggressive memory freeing you asked, so aggressive memory freeing we will
	// deliver.
	if strings.Contains(os.Getenv("GODEBUG"), "madvdontneed=1") {
		go func() {
			log.Println("Now attempting to free memory every 5s... (madvdontneed=1)")
			for range time.Tick(5 * time.Second) {
				debug.FreeOSMemory()
			}
		}()
	}
}

func main() {
	gts.Main(func() gts.MainApplication {
		var app = ui.NewApplication()

		// Load all cchat services.
		srvcs, errs := services.Get()
		if len(errs) > 0 {
			for _, err := range errs {
				log.Error(err)
			}
		}

		// Add the services.
		for _, srvc := range srvcs {
			app.AddService(srvc)
		}

		// Restore the configs.
		config.Restore()

		return app
	})

	destructor()
}
