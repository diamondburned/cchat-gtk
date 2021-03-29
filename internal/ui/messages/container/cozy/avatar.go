package cozy

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/gotk3/gotk3/gtk"
)

type Avatar struct {
	roundimage.Button
	Image *roundimage.StillImage
	url   string
}

func NewAvatar(parent primitives.Connector) *Avatar {
	img := roundimage.NewStillImage(nil, 0)
	img.SetSizeRequest(AvatarSize, AvatarSize)
	img.Show()

	avatar, _ := roundimage.NewCustomButton(img)
	avatar.SetVAlign(gtk.ALIGN_START)

	// Default icon.
	primitives.SetImageIcon(img, "user-available-symbolic", AvatarSize)

	return &Avatar{*avatar, img, ""}
}

// SetImage sets the avatar from the given label image.
func (a *Avatar) SetImage(img rich.LabelImage) {
	a.SetURL(img.URL)
}

// SetURL updates the Avatar to be that URL. It does nothing if URL is empty or
// matches the existing one.
func (a *Avatar) SetURL(url string) {
	// Check if the URL is the same. This will save us quite a few requests, as
	// some methods rely on the side-effects of other methods, and they may call
	// UpdateAuthor multiple times.
	if a.url == url || url == "" {
		return
	}

	a.url = url
	a.Image.SetImageURL(url)
}
