package input

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const AvatarSize = 20

type usernameContainer struct {
	*gtk.Revealer

	main *gtk.Box

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

// Update is not thread-safe.
func (u *usernameContainer) Update(session cchat.Session, sender cchat.ServerMessageSender) {
	// Does sender (aka Server) implement ServerNickname? If not, we fallback to
	// the username inside session.
	var err error
	if nicknamer, ok := sender.(cchat.ServerNickname); ok {
		err = errors.Wrap(nicknamer.Nickname(u), "Failed to get nickname")
	} else {
		err = errors.Wrap(session.Name(u), "Failed to get username")
	}

	// Do a bit of trivial error handling.
	if err != nil {
		log.Warn(err)
	}

	// Does session implement an icon? Update if so.
	if iconer, ok := session.(cchat.Icon); ok {
		err = iconer.Icon(u)
	}

	if err != nil {
		log.Warn(errors.Wrap(err, "Failed to get icon"))
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
