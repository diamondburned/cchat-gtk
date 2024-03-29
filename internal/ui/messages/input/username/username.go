package username

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

const AvatarSize = 24

var (
	showUser = true
	updaters config.Updaters
)

func init() {
	// Bind this revealer in settings.
	config.AppearanceAdd("Show Username in Input", config.Switch(
		&showUser,
		func(b bool) { updaters.Updated() },
	))
}

type Container struct {
	gtk.Revealer
	State *message.Author

	main   *gtk.Box
	avatar *roundimage.Image
	label  *rich.Label
}

var usernameCSS = primitives.PrepareCSS(`
	.username-view { margin: 0 5px }
`)

var fallbackAuthor = message.NewCustomAuthor("", text.Plain("self"))

func NewContainer() *Container {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.Show()

	primitives.AddClass(box, "username-view")
	primitives.AttachCSS(box, usernameCSS)

	rev, _ := gtk.RevealerNew()
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	rev.SetTransitionDuration(50)
	rev.Add(box)

	// Bind the current global revealer to this revealer for settings. This
	// operation should be thread-safe, as everything is being done in the main
	// thread.
	updaters.Add(func() { rev.SetRevealChild(showUser) })

	author := message.NewCustomAuthor("", text.Plain("self"))

	u := Container{
		Revealer: *rev,
		State:    &author,
		main:     box,
	}
	u.Reset()

	u.avatar = roundimage.NewImage(0)
	u.avatar.SetSize(AvatarSize)
	u.avatar.SetHAlign(gtk.ALIGN_CENTER)
	u.avatar.SetPlaceholderIcon("user-available-symbolic", AvatarSize)
	u.avatar.Show()

	rich.BindRoundImage(u.avatar, &u.State.Name, false)

	u.label = rich.NewLabel(&u.State.Name)
	u.label.SetEllipsize(pango.ELLIPSIZE_END)
	u.label.SetMaxWidthChars(35)
	u.label.Show()

	primitives.RemoveChildren(u.main)
	u.main.PackStart(u.avatar, false, false, 0)
	u.main.PackStart(u.label, false, false, 0)

	return &u
}

func (u *Container) SetRevealChild(reveal bool) {
	// Only reveal if showUser is true.
	u.Revealer.SetRevealChild(reveal && u.shouldReveal())
}

// shouldReveal returns whether or not the container should reveal.
func (u *Container) shouldReveal() bool {
	show := false

	show = !u.State.Name.Label().IsEmpty()
	show = show || u.avatar.GetImageURL() != ""
	show = show || showUser

	return true
}

func (u *Container) Reset() {
	u.SetRevealChild(false)
	u.State.ID = ""
	u.State.Name.Stop()
}

// Update is not thread-safe.
func (u *Container) Update(session cchat.Session, messenger cchat.Messenger) {
	u.State.ID = session.ID()
	u.SetRevealChild(true)

	if nicknamer := messenger.AsNicknamer(); nicknamer != nil {
		u.State.Name.BindNamer(u.main, "destroy", nicknamer)
	} else {
		u.State.Name.BindNamer(u.main, "destroy", session)
	}
}

// Label returns the underlying label.
func (u *Container) Label() text.Rich {
	return u.State.Name.Label()
}

// LabelMarkup returns the underlying label's markup.
func (u *Container) GetLabelMarkup() string {
	return u.label.Output().Markup
}
