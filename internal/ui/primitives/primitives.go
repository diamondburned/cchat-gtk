package primitives

import "github.com/gotk3/gotk3/gtk"

type StyleContexter interface {
	GetStyleContext() (*gtk.StyleContext, error)
}

func AddClass(styleCtx StyleContexter, classes ...string) {
	var style, _ = styleCtx.GetStyleContext()
	for _, class := range classes {
		style.AddClass(class)
	}
}

type Bin interface {
	GetChild() (gtk.IWidget, error)
}

var _ Bin = (*gtk.Bin)(nil)

func BinLeftAlignLabel(bin Bin) {
	widget, _ := bin.GetChild()
	widget.(interface{ SetHAlign(gtk.Align) }).SetHAlign(gtk.ALIGN_START)
}

func NewButtonIcon(icon string) *gtk.Image {
	img, _ := gtk.ImageNewFromIconName(icon, gtk.ICON_SIZE_BUTTON)
	return img
}

func NewImageIconPx(icon string, sizepx int) *gtk.Image {
	img, _ := gtk.ImageNew()
	SetImageIcon(img, icon, sizepx)
	return img
}

func SetImageIcon(img *gtk.Image, icon string, sizepx int) {
	img.SetProperty("icon-name", icon)
	img.SetProperty("pixel-size", sizepx)
	img.SetSizeRequest(sizepx, sizepx)
}
