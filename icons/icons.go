package icons

import (
	"log"

	"github.com/gotk3/gotk3/gdk"
)

// static assets
var assets = map[string]*gdk.Pixbuf{}

func Logo256Variant2(sz int) *gdk.Pixbuf {
	return loadPixbuf(__cchat_variant2_256, sz)
}

func Logo256(sz int) *gdk.Pixbuf {
	return loadPixbuf(__cchat_256, sz)
}

func loadPixbuf(data []byte, sz int) *gdk.Pixbuf {
	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Fatalln("Failed to create a pixbuf loader for icons:", err)
	}

	if sz > 0 {
		l.Connect("size-prepared", func() { l.SetSize(sz, sz) })
	}

	p, err := l.WriteAndReturnPixbuf(data)
	if err != nil {
		log.Fatalln("Failed to write and return pixbuf:", err)
	}

	return p
}
