package roundimage

import (
	"context"
	"log"
	"math"

	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/handy"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const (
	pi     = math.Pi
	circle = 2 * math.Pi
)

type RadiusSetter interface {
	SetRadius(float64)
}

type Connector interface {
	ConnectHandlers(connector primitives.Connector)
}

type Imager interface {
	gtk.IWidget
	RadiusSetter
	SetSizeRequest(w, h int)

	// Embed setters.
	httputil.ImageContainer

	GetPixbuf() *gdk.Pixbuf
	GetAnimation() *gdk.PixbufAnimation

	GetImage() *gtk.Image
}

// Image represents an image with abstractions for asynchronously fetching
// images from a URL as well as having interchangeable fallbacks.
type Image struct {
	gtk.Image
	Radius float64

	style *gtk.StyleContext

	procs  []imgutil.Processor
	ifNone func(context.Context)

	icon struct {
		name string
		size int
	}

	cancel context.CancelFunc
	imgURL string
	show   bool
}

var _ Imager = (*Image)(nil)

// NewImage creates a new round image. If radius is 0, then it will be half the
// dimensions. If the radius is less than 0, then nothing is rounded.
func NewImage(radius float64) *Image {
	i, err := gtk.ImageNew()
	if err != nil {
		log.Panicln("failed to create new roundimage.Image:", err)
	}

	style, _ := i.GetStyleContext()

	image := &Image{
		Image:  *i,
		Radius: radius,
		style:  style,
	}

	// Connect to the draw callback and clip the context.
	i.Connect("draw", image.drawer)

	return image
}

// NewSizedImage creates a new square image with the given square.
func NewSizedImage(radius float64, size int) *Image {
	img := NewImage(radius)
	img.SetSizeRequest(size, size)
	return img
}

// AddProcessor adds image processors that will be processed on fetched images.
// Images generated internally, such as initials, won't use it.
func (i *Image) AddProcessor(procs ...imgutil.Processor) {
	i.procs = append(i.procs, procs...)
}

// GetImage returns the underlying image widget.
func (i *Image) GetImage() *gtk.Image {
	return &i.Image
}

// Size returns the minimum side's length. This method is used when Image is
// supposed to be a square/circle.
func (i *Image) Size() int {
	w, h := i.GetSizeRequest()
	if w > h {
		return h
	}
	return w
}

// SetSIze sets the iamge's physical size. It is a convenient function for
// SetSizeRequest.
func (i *Image) SetSize(size int) {
	i.SetSizeRequest(size, size)
}

// SetIfNone sets the callback to be used if an empty URL is given to the image.
// If nil is given, then a fallback icon is used.
func (i *Image) SetIfNone(ifNone func(context.Context)) {
	i.ifNone = ifNone
}

// UpdateIfNone updates the image if the image currently does not have one
// fetched from the URL. It does nothing otherwise.
func (i *Image) UpdateIfNone() {
	if i.ifNone == nil || i.imgURL != "" {
		return
	}

	i.SetImageURL("")
}

// SetPlaceholderIcon sets the placeholder icon onto the image. The given icon
// size does not affect the image's physical size.
func (i *Image) SetPlaceholderIcon(iconName string, iconPx int) {
	i.icon.name = iconName
	i.icon.size = iconPx

	if i.imgURL == "" {
		i.SetImageURL("")
	}
}

// GetImageURL gets the image's URL. It returns an empty string if the image
// does not have a URL set.
func (i *Image) GetImageURL() string {
	return i.imgURL
}

// SetImageURL sets the image's URL. If the URL is empty, then the placeholder
// icon is used, or the IfNone callback is called, or the pixbuf is cleared.
func (i *Image) SetImageURL(url string) {
	i.SetImageURLInto(url, i)
}

// SetImageURLInto is SetImageURL, but the image container is given as an
// argument. It is used by other widgets that extend on this Image.
func (i *Image) SetImageURLInto(url string, otherImage httputil.ImageContainer) {
	i.imgURL = url

	// TODO: fix this context leak: cancel not being called on all paths.
	ctx := i.resetCtx()

	if url != "" {
		// No dynamic sizing support; yolo.
		httputil.AsyncImage(ctx, otherImage, url, i.procs...)
		return
	}

	if i.icon.name != "" {
		primitives.SetImageIcon(i, i.icon.name, i.icon.size)
		goto noImage
	}

	if i.ifNone != nil {
		i.ifNone(ctx)
		return
	}

noImage:
	i.Image.SetFromPixbuf(nil)
	i.cancel()
}

func (i *Image) resetCtx() context.Context {
	if i.cancel != nil {
		i.cancel()
		i.cancel = nil
	}

	// TODO: fix this context leak: cancel not being called on all paths.
	ctx, cancel := context.WithCancel(context.Background())
	i.cancel = cancel

	return ctx
}

// SetRadius sets the radius to be drawn with. If 0 is given, then a full circle
// is drawn, which only works best for images guaranteed to be square.
// Otherwise, the radius is either the number given or the minimum of either the
// width or height.
func (i *Image) SetRadius(r float64) {
	i.Radius = r
	i.QueueDraw()
}

func (i *Image) drawer(image *gtk.Image, cc *cairo.Context) bool {
	// Don't round if we're displaying a stock icon.
	if i.imgURL == "" && i.icon.name != "" {
		return false
	}

	a := image.GetAllocation()
	w := float64(a.GetWidth())
	h := float64(a.GetHeight())

	min := w
	// Use the largest side for radius calculation.
	if h > w {
		min = h
	}

	switch {
	// If radius is less than 0, then don't round.
	case i.Radius < 0:
		return false

	// If radius is 0, then we have to calculate our own radius.:This only
	// works if the image is a square.
	case i.Radius == 0:
		// Calculate the radius by dividing a side by 2.
		r := (min / 2)

		// Draw an arc from 0deg to 360deg.
		cc.Arc(w/2, h/2, r, 0, circle)

		// We have to do this so the arc paint doesn't leave back a black
		// background instead of the usual alpha.
		cc.SetSourceRGBA(255, 255, 255, 0)

		// Clip the image with the arc we drew.
		cc.Clip()

	// If radius is more than 0, then we have to calculate the radius from
	// the edges.
	case i.Radius > 0:
		// StackOverflow is godly.
		// https://stackoverflow.com/a/6959843.

		// Copy the variables so we can change them later.
		r := i.Radius

		// Radius should be largest a single side divided by 2.
		if max := min / 2; r > max {
			r = max
		}

		// Draw 4 arcs at 4 corners.
		cc.Arc(0+r, 0+r, r, 2*(pi/2), 3*(pi/2)) // top left
		cc.Arc(w-r, 0+r, r, 3*(pi/2), 4*(pi/2)) // top right
		cc.Arc(w-r, h-r, r, 0*(pi/2), 1*(pi/2)) // bottom right
		cc.Arc(0+r, h-r, r, 1*(pi/2), 2*(pi/2)) // bottom left

		// Close the created path.
		cc.ClosePath()
		cc.SetSourceRGBA(255, 255, 255, 0)

		// Clip the image with the arc we drew.
		cc.Clip()
	}

	// Paint the changes.
	cc.Paint()

	return false
}

// UseInitialsIfNone sets the given image to render an initial image if the
// image doesn't have a URL.
func (i *Image) UseInitialsIfNone(initialsFn func() string) {
	i.SetIfNone(func(ctx context.Context) {
		size := i.Size()
		scale := i.GetScaleFactor()

		a := handy.AvatarNew(size, initialsFn(), true)
		p := a.DrawToPixbuf(size, scale)

		if scale > 1 {
			surface, _ := gdk.CairoSurfaceCreateFromPixbuf(p, scale, nil)
			i.SetFromSurface(surface)
		} else {
			// Potentially save a copy.
			i.SetFromPixbuf(p)
		}
	})

	i.UpdateIfNone()
}
