package rich

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type IconerFn = func(context.Context, cchat.IconContainer) (func(), error)

type RoundIconContainer interface {
	gtk.IWidget

	primitives.ImageIconSetter
	roundimage.RadiusSetter

	SetImageURL(url string)

	GetStorageType() gtk.ImageType
	GetPixbuf() *gdk.Pixbuf
	GetAnimation() *gdk.PixbufAnimation
}

var (
	_ RoundIconContainer = (*roundimage.Image)(nil)
	_ RoundIconContainer = (*roundimage.StaticImage)(nil)
)

// Icon represents a rounded image container.
type Icon struct {
	*gtk.Revealer // TODO move out

	Image RoundIconContainer

	// state
	url string
}

const DefaultIconSize = 16

var _ cchat.IconContainer = (*Icon)(nil)

func NewIcon(sizepx int) *Icon {
	img, _ := roundimage.NewImage(0)
	img.Show()
	return NewCustomIcon(img, sizepx)
}

func NewCustomIcon(img RoundIconContainer, sizepx int) *Icon {
	if sizepx == 0 {
		sizepx = DefaultIconSize
	}

	rev, _ := gtk.RevealerNew()
	rev.Add(img)
	rev.SetRevealChild(false)
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	rev.SetTransitionDuration(50)

	i := &Icon{
		Revealer: rev,
		Image:    img,
	}
	i.SetSize(sizepx)

	return i
}

// URL is not thread-safe.
func (i *Icon) URL() string {
	return i.url
}

func (i *Icon) CopyPixbuf(dst httputil.ImageContainer) {
	switch i.Image.GetStorageType() {
	case gtk.IMAGE_PIXBUF:
		dst.SetFromPixbuf(i.Image.GetPixbuf())
	case gtk.IMAGE_ANIMATION:
		dst.SetFromAnimation(i.Image.GetAnimation())
	}
}

// Thread-unsafe setter methods should only be called right after construction.

// SetPlaceholderIcon is not thread-safe.
func (i *Icon) SetPlaceholderIcon(iconName string, iconSzPx int) {
	i.Image.SetRadius(-1) // square
	i.SetRevealChild(true)

	if iconName != "" {
		primitives.SetImageIcon(i.Image, iconName, iconSzPx)
	}
}

// SetSize is not thread-safe.
func (i *Icon) SetSize(szpx int) {
	i.Image.SetSizeRequest(szpx, szpx)
}

// Size returns the minimum of the image size. It is not thread-safe.
func (i *Icon) Size() int {
	w, h := i.Image.GetSizeRequest()
	if h < w {
		return h
	}
	return w
}

// SetIcon is thread-safe.
func (i *Icon) SetIcon(url string) {
	gts.ExecAsync(func() { i.SetIconUnsafe(url) })
}

func (i *Icon) AsyncSetIconer(iconer cchat.Iconer, errwrap string) {
	// Reveal to show the placeholder.
	i.SetRevealChild(true)

	// I have a hunch this will never work; as long as Go keeps a reference with
	// iconer.Icon, then destroy will never be triggered.
	ctx := primitives.HandleDestroyCtx(context.Background(), i)
	gts.Async(func() (func(), error) {
		f, err := iconer.Icon(ctx, i)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load iconer")
		}

		return func() { i.Connect("destroy", func(interface{}) { f() }) }, nil
	})
}

// SetIconUnsafe is not thread-safe.
func (i *Icon) SetIconUnsafe(url string) {
	// Setting the radius here since we resetted it for a placeholder icon.
	i.Image.SetRadius(0)
	i.SetRevealChild(true)
	i.url = url
	i.Image.SetImageURL(i.url)
}

// type EventIcon struct {
// 	*gtk.EventBox
// 	Icon *Icon
// }

// func NewEventIcon(sizepx int) *EventIcon {
// 	icn := NewIcon(sizepx)
// 	return WrapEventIcon(icn)
// }

// func WrapEventIcon(icn *Icon) *EventIcon {
// 	icn.Show()
// 	evb, _ := gtk.EventBoxNew()
// 	evb.Add(icn)

// 	return &EventIcon{
// 		EventBox: evb,
// 		Icon:     icn,
// 	}
// }

type ToggleButtonImage struct {
	gtk.ToggleButton
	Labeler
	cchat.IconContainer

	Label *gtk.Label
	Image *Icon

	Box *gtk.Box
}

var (
	_ gtk.IWidget          = (*ToggleButton)(nil)
	_ cchat.LabelContainer = (*ToggleButton)(nil)
)

func NewToggleButtonImage(content text.Rich) *ToggleButtonImage {
	img, _ := roundimage.NewStaticImage(nil, 0)
	img.Show()
	return NewCustomToggleButtonImage(img, content)
}

func NewCustomToggleButtonImage(img RoundIconContainer, content text.Rich) *ToggleButtonImage {
	l := NewLabel(content)
	l.Show()

	i := NewCustomIcon(img, 0)
	i.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(i, false, false, 0)
	box.PackStart(l, true, true, 5)
	box.Show()

	b, _ := gtk.ToggleButtonNew()
	b.Add(box)

	if connector, ok := img.(roundimage.Connector); ok {
		connector.ConnectHandlers(b)
	}

	return &ToggleButtonImage{
		ToggleButton:  *b,
		Labeler:       l, // easy inheritance of methods
		IconContainer: i,

		Label: &l.Label,
		Image: i,
		Box:   box,
	}
}
