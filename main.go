package main

import (
	"runtime"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat/services"

	// _ "github.com/diamondburned/gotk3-tcmalloc"
	// "github.com/diamondburned/gotk3-tcmalloc/heapprofiler"

	_ "github.com/diamondburned/cchat-discord"
	_ "github.com/diamondburned/cchat-mock"
)

func init() {
	go func() {
		// If you GC more, you have shorter STWs. Easy.
		for range time.Tick(10 * time.Second) {
			runtime.GC()
		}
	}()
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

		// heapprofiler.Start("/tmp/cchat-gtk")
		// gts.App.Window.Window.Connect("destroy", heapprofiler.Stop)

		return app
	})
}
