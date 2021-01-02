package username

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

const AvatarSize = 24

var showUser = true
var currentRevealer = func(bool) {} // noop by default

func init() {
	// Bind this revealer in settings.
	config.AppearanceAdd("Show Username in Input", config.Switch(
		&showUser,
		func(b bool) { currentRevealer(b) },
	))
}

type Container struct {
	*gtk.Revealer
	main   *gtk.Box
	avatar *rich.Icon
	label  *rich.Label
}

var (
	_ cchat.LabelContainer = (*Container)(nil)
	_ cchat.IconContainer  = (*Container)(nil)
)

var usernameCSS = primitives.PrepareCSS(`
	.username-view { margin: 0 5px }
`)

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
	currentRevealer = rev.SetRevealChild

	container := Container{
		Revealer: rev,
		main:     box,
	}
	container.Reset()

	return &container
}

func (u *Container) SetRevealChild(reveal bool) {
	// Only reveal if showUser is true.
	u.Revealer.SetRevealChild(reveal && u.shouldReveal())
}

// shouldReveal returns whether or not the container should reveal.
func (u *Container) shouldReveal() bool {
	return (!u.label.GetLabel().IsEmpty() || u.avatar.URL() != "") && showUser
}

func (u *Container) Reset() {
	u.SetRevealChild(false)

	u.avatar = rich.NewIcon(AvatarSize)
	u.avatar.SetPlaceholderIcon("user-available-symbolic", AvatarSize)
	u.avatar.Show()

	u.label = rich.NewLabel(text.Rich{})
	u.label.SetMaxWidthChars(35)
	u.label.Show()

	primitives.RemoveChildren(u.main)
	u.main.PackStart(u.avatar, false, false, 0)
	u.main.PackStart(u.label, false, false, 0)
}

// Update is not thread-safe.
func (u *Container) Update(session cchat.Session, messenger cchat.Messenger) {
	// Set the fallback username.
	u.label.SetLabelUnsafe(session.Name())
	// Reveal the name if it's not empty.
	u.SetRevealChild(true)

	// Does messenger implement Nicknamer? If yes, use it.
	if nicknamer := messenger.AsNicknamer(); nicknamer != nil {
		u.label.AsyncSetLabel(nicknamer.Nickname, "Error fetching server nickname")
	}

	// Does session implement an icon? Update if yes.
	if iconer := session.AsIconer(); iconer != nil {
		u.avatar.AsyncSetIconer(iconer, "Error fetching session icon URL")
	}
}

// GetLabel is not thread-safe.
func (u *Container) GetLabel() text.Rich {
	return u.label.GetLabel()
}

// GetLabelMarkup is not thread-safe.
func (u *Container) GetLabelMarkup() string {
	return u.label.Label.GetLabel()
}

// SetLabel is thread-safe.
func (u *Container) SetLabel(content text.Rich) {
	gts.ExecAsync(func() {
		u.label.SetLabelUnsafe(content)

		// Reveal if the name is not empty.
		u.SetRevealChild(true)
	})
}

// SetIcon is thread-safe.
func (u *Container) SetIcon(url string) {
	gts.ExecAsync(func() {
		u.avatar.SetIconUnsafe(url)

		// Reveal if the icon URL is not empty. We don't touch anything if the
		// URL is empty, as the name might not be.
		u.SetRevealChild(true)
	})
}

// GetIconURL is not thread-safe.
func (u *Container) GetIconURL() string {
	return u.avatar.URL()
}
