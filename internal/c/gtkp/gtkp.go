package gtkp

// #cgo pkg-config: gdk-3.0 gio-2.0 glib-2.0 gobject-2.0 gtk+-3.0
// #include <glib.h>
// #include <gtk/gtk.h>
// #include <pango/pango.h>
import "C"

import (
	"unsafe"

	"github.com/gotk3/gotk3/gtk"
)

func LabelNoHyphens(l *gtk.Label) {
	attrlist := C.pango_attr_list_new()
	defer C.pango_attr_list_unref(attrlist)

	C.pango_attr_list_insert(attrlist, C.pango_attr_insert_hyphens_new(C.FALSE))

	v := (*C.GtkLabel)(unsafe.Pointer(l.Native()))
	C.gtk_label_set_attributes(v, attrlist)
}
