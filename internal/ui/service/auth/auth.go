package auth

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/dialog"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/spinner"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type Dialog struct {
	*dialog.Dialog

	header  *gtk.HeaderBar
	backRev *gtk.Revealer
	back    *gtk.Button // might be hidden

	stack *gtk.Stack

	spinner *spinner.Boxed
	leaflet *handy.Leaflet

	stageList *StageList
	request   *RequestStack // might be nil

	Auther cchat.Authenticator
	onAuth func(cchat.Session)
}

// NewDialog makes a new authentication dialog. Auth() is called when the user
// is authenticated successfully inside the Gtk main thread.
func NewDialog(name text.Rich, authers []cchat.Authenticator, auth func(cchat.Session)) *Dialog {
	d := &Dialog{
		Auther: nil,
		onAuth: auth,
	}

	d.spinner = spinner.NewVisible()
	d.spinner.SetSizeRequest(50, 50)
	d.spinner.Stop()
	d.spinner.Show()

	d.request = NewRequestStack()
	d.request.SetHExpand(true)
	d.request.SetName("request")
	d.request.Show()

	d.leaflet = handy.LeafletNew()
	d.leaflet.SetCanSwipeBack(true)
	d.leaflet.SetCanSwipeForward(false)
	d.leaflet.SetVExpand(true)
	d.leaflet.SetTransitionType(handy.LeafletTransitionTypeSlide)
	d.leaflet.Show()

	d.stack, _ = gtk.StackNew()
	d.stack.SetVExpand(true)
	d.stack.SetHExpand(true)
	d.stack.AddNamed(d.leaflet, "leaflet")
	d.stack.AddNamed(d.spinner, "spinner")
	d.stack.SetVisibleChildName("leaflet")
	d.stack.Show()

	d.back, _ = gtk.ButtonNewFromIconName("go-previous-symbolic", gtk.ICON_SIZE_BUTTON)
	d.back.Show()
	d.back.Connect("clicked", func() {
		// If check just in case.
		if d.stageList != nil {
			d.leaflet.SetVisibleChild(d.stageList)
			d.backRev.SetRevealChild(false)
		}
	})

	d.backRev, _ = gtk.RevealerNew()
	d.backRev.SetTransitionDuration(50)
	d.backRev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	d.backRev.Add(d.back)
	d.backRev.Show()

	d.header, _ = gtk.HeaderBarNew()
	d.header.SetShowCloseButton(true)
	d.header.SetTitle("Log in to " + name.Content)
	d.header.PackStart(d.backRev)
	d.header.Show()

	d.setAuthers(authers)

	primitives.LeafletOnFold(d.leaflet, func(folded bool) {
		visibleChildName := primitives.GetName(d.leaflet.GetVisibleChild().ToWidget())

		if folded && visibleChildName == "request" {
			d.backRev.SetRevealChild(true)
		} else {
			d.backRev.SetRevealChild(false)
		}
	})

	d.Dialog = dialog.NewCSD(d.stack, d.header)
	d.Dialog.SetDefaultSize(500, 350)
	d.Dialog.Show()

	return d
}

func (d *Dialog) setAuthers(authers []cchat.Authenticator) {
	primitives.RemoveChildren(d.leaflet)

	d.request.SetRequest(nil, nil)

	d.stageList = NewStageList(authers, d.setAuther)
	d.stageList.SetName("stagelist")
	d.stageList.Show()

	d.leaflet.Add(d.stageList)
	d.leaflet.Add(d.request)

	d.stageList.SelectFirst()
	d.leaflet.SetVisibleChild(d.stageList)
	d.backRev.SetRevealChild(false)
}

func (d *Dialog) setAuther(auther cchat.Authenticator) {
	d.Auther = auther
	d.request.SetRequest(auther, d.onContinue)
	d.backRev.SetRevealChild(d.leaflet.GetFolded())
	d.leaflet.SetVisibleChild(d.request)
}

func (d *Dialog) onContinue() {
	request := d.request.Request()
	values := request.values()
	auther := d.Auther

	d.Dialog.SetSensitive(false)
	d.back.Hide()
	d.stack.SetVisibleChildName("spinner")
	d.spinner.Start()

	gts.Async(func() (func(), error) {
		s, err := auther.Authenticate(values)

		return func() {
			if err == nil {
				d.Destroy()
				d.onAuth(s)
				return
			}

			d.spinner.Stop()
			d.stack.SetVisibleChildName("leaflet")
			d.back.Show()
			d.Dialog.SetSensitive(true)

			if nextStage := err.NextStage(); nextStage != nil {
				d.setAuthers(nextStage)
			} else {
				request.SetError(err)
			}
		}, err
	})
}
