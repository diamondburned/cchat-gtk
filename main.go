package main

import (
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat/services"

	_ "github.com/diamondburned/cchat-discord"
	_ "github.com/diamondburned/cchat-mock"
)

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
}
