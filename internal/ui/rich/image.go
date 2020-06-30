package rich

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

type IconerFn = func(context.Context, cchat.IconContainer) (func(), error)

type Icon struct {
	*gtk.Revealer
	Image   *gtk.Image
	resizer imgutil.Processor
	procs   []imgutil.Processor
	size    int

	r gts.Reusable

	// state
	url string
}

const DefaultIconSize = 16

var _ cchat.IconContainer = (*Icon)(nil)

func NewIcon(sizepx int, procs ...imgutil.Processor) *Icon {
	if sizepx == 0 {
		sizepx = DefaultIconSize
	}

	img, _ := gtk.ImageNew()
	img.Show()
	img.SetSizeRequest(sizepx, sizepx)

	rev, _ := gtk.RevealerNew()
	rev.Add(img)
	rev.SetRevealChild(false)
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	rev.SetTransitionDuration(50)

	i := &Icon{
		Revealer: rev,
		Image:    img,
		procs:    procs,

		r: *gts.NewReusable(),
	}
	i.SetSize(sizepx)

	return i
}

// Reset wipes the state to be just after construction.
func (i *Icon) Reset() {
	i.url = ""
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
	i.SetRevealChild(true)
	i.SetSize(iconSzPx)

	if iconName != "" {
		primitives.SetImageIcon(i.Image, iconName, iconSzPx)
	}
}

// SetSize is not thread-safe.
func (i *Icon) SetSize(szpx int) {
	i.size = szpx
	i.Image.SetSizeRequest(szpx, szpx)
	i.resizer = imgutil.Resize(szpx, szpx)
}

// AddProcessors is not thread-safe.
func (i *Icon) AddProcessors(procs ...imgutil.Processor) {
	i.procs = append(i.procs, procs...)
}

// SetIcon is thread-safe.
func (i *Icon) SetIcon(url string) {
	gts.ExecAsync(func() { i.SetIconUnsafe(url) })
}

func (i *Icon) swapResource(v interface{}) {
	i.SetIconUnsafe(v.(*nullIcon).url)
}

func (i *Icon) AsyncSetIconer(iconer cchat.Icon, wrap string) {
	gts.AsyncUse(&i.r, i.swapResource, func(ctx context.Context) (interface{}, error) {
		ni := &nullIcon{}
		f, err := iconer.Icon(ctx, ni)
		ni.cancel = f
		return ni, err
	})
}

// SetIconUnsafe is not thread-safe.
func (i *Icon) SetIconUnsafe(url string) {
	i.SetRevealChild(true)
	i.url = url
	i.updateAsync()
}

func (i *Icon) updateAsync() {
	httputil.AsyncImage(i.Image, i.url, imgutil.Prepend(i.resizer, i.procs)...)
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

	i := NewIcon(0)
	i.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(i, false, false, 0)
	box.PackStart(l, true, true, 5)
	box.Show()

	b, _ := gtk.ToggleButtonNew()
	b.Add(box)

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
