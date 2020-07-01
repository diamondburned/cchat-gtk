package typing

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/username"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type State struct {
	typers []cchat.Typer
}

func NewState() *State {
	return &State{}
}

func (s *State) Empty() bool {
	// return len(s.typers) == 0
	return false
}

var typingIndicatorCSS = primitives.PrepareCSS(`
	.typing-indicator {
		border-radius: 8px 8px 0 0;
		color: alpha(@theme_fg_color, 0.8);
		background-color: @theme_base_color;
	}
`)

var smallfonts = primitives.PrepareCSS(`
	* { font-size: 0.9em; }
`)

type Container struct {
	*gtk.Revealer
	empty bool // && state.Empty()
	State *State
}

const placeholder = "Bruh moment..."

func New() *Container {
	d := NewDots()
	d.Show()

	l, _ := gtk.LabelNew(placeholder)
	l.SetXAlign(0)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.Show()
	primitives.AttachCSS(l, smallfonts)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.PackStart(d, false, false, username.VMargin)
	b.PackStart(l, true, true, 0)
	b.SetMarginStart(username.VMargin * 2)
	b.SetMarginEnd(username.VMargin * 2)
	b.Show()

	r, _ := gtk.RevealerNew()
	r.SetTransitionDuration(50)
	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_CROSSFADE)
	r.SetRevealChild(true)
	r.Add(b)

	state := NewState()

	primitives.AddClass(b, "typing-indicator")
	primitives.AttachCSS(b, typingIndicatorCSS)

	return &Container{
		Revealer: r,
		State:    state,
	}
}
