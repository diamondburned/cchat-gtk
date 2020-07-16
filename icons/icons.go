package icons

import (
	"bytes"
	"log"

	"github.com/gotk3/gotk3/gdk"
	"github.com/markbates/pkger"
)

// static assets
var assets = map[string]*gdk.Pixbuf{}

func Logo256Variant2() *gdk.Pixbuf {
	return loadPixbuf(__cchat_variant2_256)
}

func Logo256() *gdk.Pixbuf {
	return loadPixbuf(__cchat_256)
}

func loadPixbuf(data []byte) *gdk.Pixbuf {
	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Fatalln("Failed to create a pixbuf loader for icons:", err)
	}

	p, err := l.WriteAndReturnPixbuf(data)
	if err != nil {
		log.Fatalln("Failed to write and return pixbuf:", err)
	}

	return p
}

func readFile(name string) []byte {
	f, err := pkger.Open(name)
	if err != nil {
		log.Fatalln("Failed to open pkger file:", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(f); err != nil {
		log.Fatalln("Failed to read from pkger file:", err)
	}

	return buf.Bytes()
}
