package labelutils

// #cgo pkg-config: gdk-3.0 gio-2.0 glib-2.0 gobject-2.0 gtk+-3.0
// #include <glib.h>
// #include <gtk/gtk.h>
// #include <pango/pango.h>
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

func gbool(b bool) C.gboolean {
	if b {
		return C.gboolean(C.TRUE)
	} else {
		return C.gboolean(C.FALSE)
	}
}

type Attribute = func() *C.PangoAttribute

func InsertHyphens(hyphens bool) Attribute {
	return func() *C.PangoAttribute {
		return C.pango_attr_insert_hyphens_new(gbool(hyphens))
	}
}

func Scale(factor float64) Attribute {
	return func() *C.PangoAttribute {
		return C.pango_attr_scale_new(C.double(factor))
	}
}

func Underline(underline pango.Underline) Attribute {
	return func() *C.PangoAttribute {
		return C.pango_attr_underline_new(C.PangoUnderline(underline))
	}
}

func Strikethrough(strikethrough bool) Attribute {
	return func() *C.PangoAttribute {
		return C.pango_attr_strikethrough_new(gbool(strikethrough))
	}
}

const u16divu8 = 65535 / 255

func rgb(hex uint32) (r, g, b uint16) {
	r = uint16(hex>>16&255) * u16divu8
	g = uint16(hex>>8&255) * u16divu8
	b = uint16(hex&255) * u16divu8
	return
}

func Background(hex uint32) Attribute {
	r, g, b := rgb(hex)

	return func() *C.PangoAttribute {
		return C.pango_attr_background_new(C.guint16(r), C.guint16(g), C.guint16(b))
	}
}

func Foreground(hex uint32) Attribute {
	r, g, b := rgb(hex)

	return func() *C.PangoAttribute {
		return C.pango_attr_foreground_new(C.guint16(r), C.guint16(g), C.guint16(b))
	}
}

func Style(style pango.Style) Attribute {
	return func() *C.PangoAttribute {
		return C.pango_attr_style_new(C.PangoStyle(style))
	}
}

func Family(family string) Attribute {
	return func() *C.PangoAttribute {
		str := C.CString(family)
		defer C.free(unsafe.Pointer(str))
		return C.pango_attr_family_new(str)
	}
}

func AddAttr(l *gtk.Label, attrs ...Attribute) {
	attrlist := C.pango_attr_list_new()
	defer C.pango_attr_list_unref(attrlist)

	for _, attr := range attrs {
		// attr() should not unref; insert is transfer-full
		// https://discourse.gnome.org/t/pango-how-to-turn-off-hyphenation-for-char-wrapping/2101/2
		C.pango_attr_list_insert(attrlist, attr())
	}

	v := (*C.GtkLabel)(unsafe.Pointer(l.Native()))
	C.gtk_label_set_attributes(v, attrlist)
}
