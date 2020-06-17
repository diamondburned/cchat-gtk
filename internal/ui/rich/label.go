package rich

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type Labeler interface {
	// thread-safe
	cchat.LabelContainer // thread-safe

	// not thread-safe
	SetLabelUnsafe(text.Rich)
	GetLabel() text.Rich
	GetText() string
	Reset()
}

type Label struct {
	gtk.Label
	current text.Rich

	// Reusable primitive.
	r gts.Reusable
}

var (
	_ gtk.IWidget = (*Label)(nil)
	_ Labeler     = (*Label)(nil)
)

func NewLabel(content text.Rich) *Label {
	label, _ := gtk.LabelNew("")
	label.SetMarkup(parser.RenderMarkup(content))
	label.SetXAlign(0) // left align
	label.SetEllipsize(pango.ELLIPSIZE_END)

	return &Label{
		Label:   *label,
		current: content,
		// reusable primitive, take reference
		r: *gts.NewReusable(),
	}
}

// Reset wipes the state to be just after construction.
func (l *Label) Reset() {
	l.current = text.Rich{}
	l.r.Invalidate()
	l.Label.SetText("")
}

// swapResource is reserved for internal use only.
func (l *Label) swapResource(v interface{}) {
	l.SetLabelUnsafe(v.(*nullLabel).Rich)
}

func (l *Label) AsyncSetLabel(fn func(context.Context, cchat.LabelContainer) error, info string) {
	gts.AsyncUse(&l.r, l.swapResource, func(ctx context.Context) (interface{}, error) {
		var nl = &nullLabel{}
		return nl, fn(ctx, nl)
	})
}

// SetLabel is thread-safe.
func (l *Label) SetLabel(content text.Rich) {
	gts.ExecAsync(func() { l.SetLabelUnsafe(content) })
}

// SetLabelUnsafe sets the label in the current thread, meaning it's not
// thread-safe.
func (l *Label) SetLabelUnsafe(content text.Rich) {
	l.current = content
	l.SetMarkup(parser.RenderMarkup(content))
}

// GetLabel is NOT thread-safe.
func (l *Label) GetLabel() text.Rich {
	return l.current
}

// GetText is NOT thread-safe.
func (l *Label) GetText() string {
	return l.current.Content
}

type ToggleButton struct {
	gtk.ToggleButton
	Label
}

var (
	_ gtk.IWidget          = (*ToggleButton)(nil)
	_ cchat.LabelContainer = (*ToggleButton)(nil)
)

func NewToggleButton(content text.Rich) *ToggleButton {
	l := NewLabel(content)
	l.Show()

	b, _ := gtk.ToggleButtonNew()
	primitives.BinLeftAlignLabel(b)

	b.Add(l)

	return &ToggleButton{*b, *l}
}
