package rich

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// LabelRenderer is the input/output function to render a rich text segment to
// Pango markup.
type LabelRenderer = func(text.Rich) markup.RenderOutput

// RenderSkipImages is a label renderer that skips images.
func RenderSkipImages(rich text.Rich) markup.RenderOutput {
	return markup.RenderCmplxWithConfig(rich, markup.RenderConfig{
		SkipImages:     true,
		NoMentionLinks: true,
	})
}

// Label provides an abstraction around a regular GTK label that can be
// self-updated. Widgets that extend off of this (such as ToggleButton) does not
// need to manually
type Label struct {
	gtk.Label
	label  text.Rich
	output markup.RenderOutput
	render LabelRenderer
}

var _ gtk.IWidget = (*Label)(nil)

// NewStaticLabel creates a static, non-updating label.
func NewStaticLabel(rich text.Rich) *Label {
	label, _ := gtk.LabelNew("")
	label.SetXAlign(0) // left align
	label.SetEllipsize(pango.ELLIPSIZE_END)

	if !rich.IsEmpty() {
		label.SetMarkup(markup.Render(rich))
	}

	return &Label{Label: *label}
}

// NewLabel creates a self-updating label.
func NewLabel(state LabelStateStorer) *Label {
	return NewLabelWithRenderer(state, nil)
}

// NewLabelWithRenderer creates a self-updating label using the given renderer.
func NewLabelWithRenderer(state LabelStateStorer, r LabelRenderer) *Label {
	l := NewStaticLabel(text.Plain(""))
	l.render = r
	state.OnUpdate(func() { l.SetLabel(state.Label()) })
	return l
}

// Output returns the rendered output.
func (l *Label) Output() markup.RenderOutput {
	return l.output
}

// SetLabel sets the label in the current thread, meaning it's not thread-safe.
func (l *Label) SetLabel(content text.Rich) {
	// Save a call if the content is empty.
	if content.IsEmpty() {
		l.label = content
		l.output = markup.RenderOutput{}

		return
	}

	l.label = content

	var out markup.RenderOutput
	if l.render != nil {
		out = l.render(content)
	} else {
		out = markup.RenderCmplx(content)
	}

	l.output = out
	l.SetMarkup(out.Markup)
	l.SetTooltipMarkup(out.Markup)
}

// SetRenderer sets a custom renderer. If the given renderer is nil, then the
// default markup renderer is used instead. The label is automatically updated.
func (l *Label) SetRenderer(renderer LabelRenderer) {
	l.render = renderer
	l.SetLabel(l.label)
}
