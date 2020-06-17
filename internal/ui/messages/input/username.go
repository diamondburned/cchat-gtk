package input

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

const AvatarSize = 20

type usernameContainer struct {
	*gtk.Revealer
	main   *gtk.Box
	avatar *rich.Icon
	label  *rich.Label
}

var (
	_ cchat.LabelContainer = (*usernameContainer)(nil)
	_ cchat.IconContainer  = (*usernameContainer)(nil)
)

func newUsernameContainer() *usernameContainer {
	avatar := rich.NewIcon(AvatarSize, imgutil.Round(true))
	avatar.SetPlaceholderIcon("user-available-symbolic", AvatarSize)
	avatar.Show()

	label := rich.NewLabel(text.Rich{})
	label.SetMaxWidthChars(35)
	label.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.PackStart(avatar, false, false, 0)
	box.PackStart(label, false, false, 0)
	box.SetMarginStart(10)
	box.SetMarginEnd(10)
	box.SetMarginTop(inputmargin)
	box.SetMarginBottom(inputmargin)
	box.SetVAlign(gtk.ALIGN_START)
	box.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	rev.SetTransitionDuration(50)
	rev.Add(box)

	return &usernameContainer{
		Revealer: rev,
		main:     box,
		avatar:   avatar,
		label:    label,
	}
}

func (u *usernameContainer) Reset() {
	u.SetRevealChild(false)
	u.avatar.Reset()
	u.label.Reset()
}

// Update is not thread-safe.
func (u *usernameContainer) Update(session cchat.Session, sender cchat.ServerMessageSender) {
	// Set the fallback username.
	u.label.SetLabelUnsafe(session.Name())
	// Reveal the name if it's not empty.
	u.SetRevealChild(!u.label.GetLabel().Empty())

	// Does sender (aka Server) implement ServerNickname? If yes, use it.
	if nicknamer, ok := sender.(cchat.ServerNickname); ok {
		u.label.AsyncSetLabel(nicknamer.Nickname, "Error fetching server nickname")
	}

	// Does session implement an icon? Update if yes.
	if iconer, ok := session.(cchat.Icon); ok {
		u.avatar.AsyncSetIcon(iconer.Icon, "Error fetching session icon URL")
	}
}

// GetLabel is not thread-safe.
func (u *usernameContainer) GetLabel() text.Rich {
	return u.label.GetLabel()
}

// SetLabel is thread-safe.
func (u *usernameContainer) SetLabel(content text.Rich) {
	gts.ExecAsync(func() {
		u.label.SetLabelUnsafe(content)

		// Reveal if the name is not empty.
		u.SetRevealChild(!u.label.GetLabel().Empty())
	})
}

// SetIcon is thread-safe.
func (u *usernameContainer) SetIcon(url string) {
	gts.ExecAsync(func() {
		u.avatar.SetIconUnsafe(url)

		// Reveal if the icon URL is not empty. We don't touch anything if the
		// URL is empty, as the name might not be.
		if url != "" {
			u.SetRevealChild(true)
		}
	})
}

// GetIconURL is not thread-safe.
func (u *usernameContainer) GetIconURL() string {
	return u.avatar.URL()
}
