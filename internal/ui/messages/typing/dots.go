package typing

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

var dotsCSS = primitives.PrepareCSS(`
	@keyframes breathing {
		0% {   opacity: 0.66; }
		100% { opacity: 0.12; }
	}

	label {
		animation: breathing 800ms infinite alternate;
	}

	label:nth-child(1) {
		animation-delay: 000ms;
	}

	label:nth-child(2) {
		animation-delay: 150ms;
	}

	label:nth-child(3) {
		animation-delay: 300ms;
	}
`)

const breathingChar = "●"

func NewDots() *gtk.Box {
	c1, _ := gtk.LabelNew(breathingChar)
	c1.Show()
	c2, _ := gtk.LabelNew(breathingChar)
	c2.Show()
	c3, _ := gtk.LabelNew(breathingChar)
	c3.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.Add(c1)
	b.Add(c2)
	b.Add(c3)

	primitives.AddClass(b, "breathing-dots")

	primitives.AttachCSS(c1, dotsCSS)
	primitives.AttachCSS(c1, smallfonts)
	primitives.AttachCSS(c2, dotsCSS)
	primitives.AttachCSS(c2, smallfonts)
	primitives.AttachCSS(c3, dotsCSS)
	primitives.AttachCSS(c3, smallfonts)

	return b
}
