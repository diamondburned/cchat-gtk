package icons

import (
	"log"

	_ "embed"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
)

// static assets
// var assets = map[string]*gdk.Pixbuf{}

//go:embed cchat_256.png
var __cchat_256 []byte

//go:embed cchat-variant2_256.png
var __cchat_variant2_256 []byte

func Logo256Variant2(sz, scale int) *cairo.Surface {
	return mustSurface(loadPixbuf(__cchat_variant2_256, sz, scale), scale)
}

func Logo256(sz, scale int) *cairo.Surface {
	return mustSurface(loadPixbuf(__cchat_256, sz, scale), scale)
}

func Logo256Pixbuf() *gdk.Pixbuf {
	return loadPixbuf(__cchat_256, 256, 1)
}

func mustSurface(p *gdk.Pixbuf, scale int) *cairo.Surface {
	surface, err := gdk.CairoSurfaceCreateFromPixbuf(p, scale, nil)
	if err != nil {
		log.Fatalln("Failed to create surface from pixbuf:", err)
	}
	return surface
}

func loadPixbuf(data []byte, sz, scale int) *gdk.Pixbuf {
	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Fatalln("Failed to create a pixbuf loader for icons:", err)
	}

	if sz > 0 {
		l.Connect("size-prepared", func(l *gdk.PixbufLoader) {
			l.SetSize(sz*scale, sz*scale)
		})
	}

	p, err := l.WriteAndReturnPixbuf(data)
	if err != nil {
		log.Fatalln("Failed to write and return pixbuf:", err)
	}

	return p
}
