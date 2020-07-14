package typing

import (
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var typingIndicatorCSS = primitives.PrepareCSS(`
	.typing-indicator {
		margin: 0 6px;
		border-radius: 6px 6px 0 0;
		color: alpha(@theme_fg_color, 0.8);
		background-color: @theme_base_color;
	}
`)

var smallfonts = primitives.PrepareCSS(`
	* { font-size: 0.9em; }
`)

type Container struct {
	*gtk.Revealer
	state *State
}

func New() *Container {
	d := NewDots()
	d.Show()

	l, _ := gtk.LabelNew("")
	l.SetXAlign(0)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.Show()
	primitives.AttachCSS(l, smallfonts)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.PackStart(d, false, false, 4)
	b.PackStart(l, true, true, 0)
	b.Show()

	r, _ := gtk.RevealerNew()
	r.SetTransitionDuration(50)
	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_CROSSFADE)
	r.SetRevealChild(false)
	r.Add(b)

	primitives.AddClass(b, "typing-indicator")
	primitives.AttachCSS(b, typingIndicatorCSS)

	state := NewState(func(s *State, empty bool) {
		r.SetRevealChild(!empty)
		l.SetMarkup(render(s.typers))
	})

	// On label destroy, stop the state loop as well.
	l.Connect("destroy", state.stopper)

	return &Container{
		Revealer: r,
		state:    state,
	}
}

func (c *Container) Reset() {
	c.state.reset()
	c.SetRevealChild(false)
}

func (c *Container) RemoveAuthor(author cchat.MessageAuthor) {
	c.state.removeTyper(author.ID())
}

func (c *Container) TrySubscribe(svmsg cchat.ServerMessage) bool {
	ti, ok := svmsg.(cchat.ServerMessageTypingIndicator)
	if !ok {
		return false
	}

	c.state.Subscribe(ti)
	return true
}

func render(typers []cchat.Typer) string {
	// fast path
	if len(typers) == 0 {
		return ""
	}

	var builder strings.Builder

	for i, typer := range typers {
		builder.WriteString("<b>")
		builder.WriteString(markup.Render(typer.Name()))
		builder.WriteString("</b>")

		switch i {
		case len(typers) - 2:
			builder.WriteString(" and ")
		case len(typers) - 1:
			// Write nothing if this is the last item.
		default:
			builder.WriteString(", ")
		}
	}

	if len(typers) == 1 {
		builder.WriteString(" is typing.")
	} else {
		builder.WriteString(" are typing.")
	}

	return builder.String()
}
