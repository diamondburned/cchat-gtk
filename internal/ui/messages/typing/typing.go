package typing

import (
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var typingIndicatorCSS = primitives.PrepareClassCSS("typing-indicator", `
	.typing-indicator {
		margin:   0 6px;
		margin-top: 2px;
		padding:  0 4px;

		border-radius: 6px 6px 0 0;

		color: alpha(@theme_fg_color, 0.8);
		background-color: @theme_base_color;
	}
`)

var typingLabelCSS = primitives.PrepareClassCSS("typing-label", `
	.typing-label {
		padding-left: 2px;
	}
`)

var smallfonts = primitives.PrepareCSS(`
	* { font-size: 0.9em; }
`)

const (
	// Keep the same as input.
	ClampMaxSize   = 1000 - 6*2 // account for margin
	ClampThreshold = ClampMaxSize
)

type Container struct {
	*gtk.Revealer
	state *State

	clamp *handy.Clamp
	dots  *gtk.Box
	label *gtk.Label

	// borrow, if true, will not update the label until it is set to false.
	borrow bool
	// markup stores the label if the label view is not borrowed.
	markup string
}

func New() *Container {
	d := NewDots()
	d.Show()

	l, _ := gtk.LabelNew("")
	l.SetXAlign(0)
	l.SetEllipsize(pango.ELLIPSIZE_END)
	l.Show()
	typingLabelCSS(l)
	primitives.AttachCSS(l, smallfonts)

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.PackStart(d, false, false, 0)
	b.PackStart(l, true, true, 0)
	b.Show()

	c := handy.ClampNew()
	c.SetMaximumSize(ClampMaxSize)
	c.SetTighteningThreshold(ClampThreshold)
	c.SetHExpand(true)
	c.Add(b)
	c.Show()

	r, _ := gtk.RevealerNew()
	r.SetTransitionDuration(100)
	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_CROSSFADE)
	r.SetRevealChild(false)
	r.Add(c)

	typingIndicatorCSS(b)

	container := &Container{
		Revealer: r,
		dots:     d,
		label:    l,
	}

	container.state = NewState(func(s *State, empty bool) {
		if !empty {
			container.markup = render(s.typers)
		} else {
			container.markup = ""
		}

		if !container.borrow {
			r.SetRevealChild(!empty)
			l.SetMarkup(container.markup)
		}
	})

	// On label destroy, stop the state loop as well.
	l.Connect("destroy", func(interface{}) { container.state.stopper() })

	return container
}

func (c *Container) Reset() {
	c.state.reset()
	c.SetRevealChild(false)
}

// BorrowLabel borrows the container label. The typing indicator will display
// the given markup string instead of the markup it is intended to display until
// Unborrow is called.
func (c *Container) BorrowLabel(markup string) {
	c.borrow = true
	c.label.SetMarkup(markup)
	c.dots.Hide() // bad, TODO use revealer
	c.SetRevealChild(true)
}

// Unborrow stops borrowing the typing indicator, returning it to the state it
// is supposed to show. Calling Unborrow multiple times will only take effect
// for the first time.
func (c *Container) Unborrow() {
	if c.borrow {
		c.label.SetMarkup(c.markup)
		c.SetRevealChild(c.markup != "")
		c.dots.Show() // bad, TODO use revealer
		c.borrow = false
	}
}

func (c *Container) RemoveAuthor(author cchat.Author) {
	c.state.removeTyper(author.ID())
}

func (c *Container) TrySubscribe(svmsg cchat.Messenger) bool {
	var tindicator = svmsg.AsTypingIndicator()
	if tindicator == nil {
		return false
	}

	c.state.Subscribe(tindicator)
	return true
}

var noMentionLinks = markup.RenderConfig{
	NoMentionLinks: true,
}

func render(typers []cchat.Typer) string {
	// fast path
	if len(typers) == 0 {
		return ""
	}

	var builder strings.Builder

	for i, typer := range typers {
		output := markup.RenderCmplxWithConfig(typer.Name(), noMentionLinks)

		builder.WriteString("<b>")
		builder.WriteString(output.Markup)
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
