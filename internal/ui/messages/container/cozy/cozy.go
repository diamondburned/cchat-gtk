package cozy

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/autoscroll"
	"github.com/gotk3/gotk3/gtk"
)

type Container struct {
	*autoscroll.ScrolledWindow
	main      *gtk.Grid
	messages  map[string]Message
	nonceMsgs map[string]Message
}
