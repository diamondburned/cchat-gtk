package icons

import (
	"bytes"
	"log"

	"github.com/gotk3/gotk3/gdk"
	"github.com/markbates/pkger"
)

// static assets
var logo256 *gdk.Pixbuf

func Logo256() *gdk.Pixbuf {
	if logo256 == nil {
		logo256 = loadPixbuf(pkger.Include("/icons/cchat-variant2_256.png"))
	}
	return logo256
}

func loadPixbuf(name string) *gdk.Pixbuf {
	l, err := gdk.PixbufLoaderNew()
	if err != nil {
		log.Fatalln("Failed to create a pixbuf loader for icons:", err)
	}

	p, err := l.WriteAndReturnPixbuf(readFile(name))
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
