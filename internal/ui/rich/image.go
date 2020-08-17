package rich

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type IconerFn = func(context.Context, cchat.IconContainer) (func(), error)

type RoundIconContainer interface {
	gtk.IWidget
	httputil.ImageContainer
	primitives.ImageIconSetter
	roundimage.RadiusSetter

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
	*gtk.Revealer
	Image RoundIconContainer
	procs []imgutil.Processor
	size  int

	r *Reusable

	// state
	url string
}

const DefaultIconSize = 16

var _ cchat.IconContainer = (*Icon)(nil)

func NewIcon(sizepx int, procs ...imgutil.Processor) *Icon {
	img, _ := roundimage.NewImage(0)
	img.Show()
	return NewCustomIcon(img, sizepx, procs...)
}

func NewCustomIcon(img RoundIconContainer, sizepx int, procs ...imgutil.Processor) *Icon {
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
		procs:    procs,
	}
	i.SetSize(sizepx)
	i.r = NewReusable(func(ni *nullIcon) {
		i.SetIconUnsafe(ni.url)
	})

	return i
}

// Reset wipes the state to be just after construction.
func (i *Icon) Reset() {
	i.url = ""
	i.r.Invalidate() // invalidate async fetching images
	i.Revealer.SetRevealChild(false)
	i.Image.SetFromPixbuf(nil) // destroy old pb
}

// URL is not thread-safe.
func (i *Icon) URL() string {
	return i.url
}

func (i *Icon) Size() int {
	return i.size
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
	i.size = szpx
	i.Image.SetSizeRequest(szpx, szpx)
}

// AddProcessors is not thread-safe.
func (i *Icon) AddProcessors(procs ...imgutil.Processor) {
	i.procs = append(i.procs, procs...)
}

// SetIcon is thread-safe.
func (i *Icon) SetIcon(url string) {
	gts.ExecAsync(func() { i.SetIconUnsafe(url) })
}

func (i *Icon) AsyncSetIconer(iconer cchat.Icon, errwrap string) {
	AsyncUse(i.r, func(ctx context.Context) (interface{}, func(), error) {
		ni := &nullIcon{}
		f, err := iconer.Icon(ctx, ni)
		return ni, f, errors.Wrap(err, errwrap)
	})
}

// SetIconUnsafe is not thread-safe.
func (i *Icon) SetIconUnsafe(url string) {
	i.Image.SetRadius(0) // round
	i.SetRevealChild(true)
	i.url = url
	i.updateAsync()
}

func (i *Icon) updateAsync() {
	httputil.AsyncImageSized(i.Image, i.url, i.size, i.size, i.procs...)
}

type EventIcon struct {
	*gtk.EventBox
	Icon *Icon
}

func NewEventIcon(sizepx int, pp ...imgutil.Processor) *EventIcon {
	icn := NewIcon(sizepx, pp...)
	icn.Show()

	evb, _ := gtk.EventBoxNew()
	evb.Add(icn)

	return &EventIcon{
		EventBox: evb,
		Icon:     icn,
	}
}

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
	l := NewLabel(content)
	l.Show()

	img, _ := roundimage.NewStaticImage(nil, 0)
	img.Show()
	i := NewCustomIcon(img, 0)
	i.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(i, false, false, 0)
	box.PackStart(l, true, true, 5)
	box.Show()

	b, _ := gtk.ToggleButtonNew()
	b.Add(box)

	img.ConnectHandlers(b)

	return &ToggleButtonImage{
		ToggleButton:  *b,
		Labeler:       l, // easy inheritance of methods
		IconContainer: i,

		Label: &l.Label,
		Image: i,
		Box:   box,
	}
}

func (t *ToggleButtonImage) Reset() {
	t.Labeler.Reset()
	t.Image.Reset()
}
