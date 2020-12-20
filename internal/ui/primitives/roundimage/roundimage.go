package roundimage

import (
	"context"
	"math"

	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
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

type Image struct {
	*gtk.Image
	Radius float64
	procs  []imgutil.Processor
}

var _ Imager = (*Image)(nil)

// NewImage creates a new round image. If radius is 0, then it will be half the
// dimensions. If the radius is less than 0, then nothing is rounded.
func NewImage(radius float64) (*Image, error) {
	i, err := gtk.ImageNew()
	if err != nil {
		return nil, err
	}

	image := &Image{Image: i, Radius: radius}

	// Connect to the draw callback and clip the context.
	i.Connect("draw", image.drawer)

	return image, nil
}

func (i *Image) AddProcessor(procs ...imgutil.Processor) {
	i.procs = append(i.procs, procs...)
}

func (i *Image) GetImage() *gtk.Image {
	return i.Image
}

func (i *Image) SetImageURL(url string) {
	// No dynamic sizing support; yolo.
	httputil.AsyncImage(context.Background(), i, url, i.procs...)
}

func (i *Image) SetRadius(r float64) {
	i.Radius = r
}

func (i *Image) drawer(widget gtk.IWidget, cc *cairo.Context) bool {
	var w = float64(i.GetAllocatedWidth())
	var h = float64(i.GetAllocatedHeight())

	var min = w
	// Use the smallest side for radius calculation.
	if h < w {
		min = h
	}

	// Copy the variables in case we need to change them.
	var r = i.Radius

	switch {
	// If radius is less than 0, then don't round.
	case r < 0:
		return false

	// If radius is 0, then we have to calculate our own radius.:This only
	// works if the image is a square.
	case r == 0:
		// Calculate the radius by dividing a side by 2.
		r = (min / 2)

		// Draw an arc from 0deg to 360deg.
		cc.Arc(w/2, h/2, r, 0, circle)

		// We have to do this so the arc paint doesn't leave back a black
		// background instead of the usual alpha.
		cc.SetSourceRGBA(255, 255, 255, 0)

		// Clip the image with the arc we drew.
		cc.Clip()

	// If radius is more than 0, then we have to calculate the radius from
	// the edges.
	case r > 0:
		// StackOverflow is godly.
		// https://stackoverflow.com/a/6959843.

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
