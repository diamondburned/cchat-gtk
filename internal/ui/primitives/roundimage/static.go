package roundimage

import (
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gdk"
)

// StaticImage is an image that only plays a GIF if it's hovered on top of.
type StaticImage struct {
	*Image
	animation *gdk.PixbufAnimation
}

var (
	_ Imager                  = (*StaticImage)(nil)
	_ Connector               = (*StaticImage)(nil)
	_ httputil.ImageContainer = (*StaticImage)(nil)
)

func NewStaticImage(parent primitives.Connector, radius float64) (*StaticImage, error) {
	i, err := NewImage(radius)
	if err != nil {
		return nil, err
	}

	var s = &StaticImage{i, nil}
	if parent != nil {
		s.ConnectHandlers(parent)
	}

	return s, nil
}

func (s *StaticImage) ConnectHandlers(connector primitives.Connector) {
	connector.Connect("enter-notify-event", func() {
		if s.animation != nil {
			s.Image.SetFromAnimation(s.animation)
		}
	})
	connector.Connect("leave-notify-event", func() {
		if s.animation != nil {
			s.Image.SetFromPixbuf(s.animation.GetStaticImage())
		}
	})
}

func (s *StaticImage) SetFromPixbuf(pb *gdk.Pixbuf) {
	s.animation = nil
	s.Image.SetFromPixbuf(pb)
}

func (s *StaticImage) SetFromAnimation(anim *gdk.PixbufAnimation) {
	s.animation = anim
	s.Image.SetFromPixbuf(anim.GetStaticImage())
}

func (s *StaticImage) GetAnimation() *gdk.PixbufAnimation {
	return s.animation
}
