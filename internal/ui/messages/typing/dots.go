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
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	primitives.AddClass(b, "breathing-dots")

	for i := 0; i < 3; i++ {
		c, _ := gtk.LabelNew(breathingChar)
		c.Show()

		primitives.AttachCSS(c, dotsCSS)
		primitives.AttachCSS(c, smallfonts)

		b.Add(c)
	}

	return b
}
