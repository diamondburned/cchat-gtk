package roundimage

import (
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// TODO: GIF support

// TextSetter is an interface for setting texts.
type TextSetter interface {
	SetText(text string)
}

func TrySetText(imager Imager, text string) {
	if setter, ok := imager.(TextSetter); ok {
		setter.SetText(text)
	}
}

// Avatar is a static HdyAvatar container.
type Avatar struct {
	handy.Avatar
	pixbuf *gdk.Pixbuf
	size   int
}

// Make a better API that allows scaling.

var (
	_ Imager                  = (*Avatar)(nil)
	_ TextSetter              = (*Avatar)(nil)
	_ httputil.ImageContainer = (*Avatar)(nil)
)

func NewAvatar(size int) *Avatar {
	avatar := Avatar{
		Avatar: *handy.AvatarNew(size, "", true),
		size:   size,
	}
	// Set the load function. This should hopefully trigger a reload.
	avatar.SetImageLoadFunc(avatar.loadFunc)

	return &avatar
}

func (a *Avatar) GetSizeRequest() (int, int) {
	return a.size, a.size
}

// SetSizeRequest sets the avatar size. The actual size is min(w, h).
func (a *Avatar) SetSizeRequest(w, h int) {
	var min = w
	if w > h {
		min = h
	}

	a.Avatar.SetSize(min)
	a.Avatar.SetSizeRequest(w, h)
}

func (a *Avatar) loadFunc(size int) *gdk.Pixbuf {
	if a.pixbuf == nil {
		a.size = size
		return nil
	}

	if a.size != size {
		a.size = size

		p, err := a.pixbuf.ScaleSimple(size, size, gdk.INTERP_HYPER)
		if err != nil {
			return a.pixbuf
		}

		a.pixbuf = p
	}

	return a.pixbuf
}

// SetRadius is a no-op.
func (a *Avatar) SetRadius(float64) {}

// SetFromPixbuf sets the pixbuf.
func (a *Avatar) SetFromPixbuf(pb *gdk.Pixbuf) {
	a.pixbuf = pb
	a.Avatar.SetImageLoadFunc(a.loadFunc)
}

func (a *Avatar) SetFromAnimation(pa *gdk.PixbufAnimation) {
	a.pixbuf = pa.GetStaticImage()
	a.Avatar.SetImageLoadFunc(a.loadFunc)
}

func (a *Avatar) GetPixbuf() *gdk.Pixbuf {
	return a.pixbuf
}

// GetAnimation returns nil.
func (a *Avatar) GetAnimation() *gdk.PixbufAnimation {
	return nil
}

// GetImage returns nil.
func (a *Avatar) GetImage() *gtk.Image {
	return nil
}

func (a *Avatar) GetStorageType() gtk.ImageType {
	return gtk.IMAGE_PIXBUF
}
