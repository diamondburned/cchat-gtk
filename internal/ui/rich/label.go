package rich

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
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

// SuperLabeler represents a label that inherits the current labeler.
type SuperLabeler interface {
	SetLabelUnsafe(text.Rich)
	Reset()
}

type LabelerFn = func(context.Context, cchat.LabelContainer) (func(), error)

type Label struct {
	gtk.Label
	Current text.Rich

	// Reusable primitive.
	r *Reusable

	// super unexported field for inheritance
	super SuperLabeler
}

var (
	_ gtk.IWidget = (*Label)(nil)
	_ Labeler     = (*Label)(nil)
)

func NewLabel(content text.Rich) *Label {
	label, _ := gtk.LabelNew("")
	label.SetMarkup(markup.Render(content))
	label.SetXAlign(0) // left align
	label.SetEllipsize(pango.ELLIPSIZE_END)

	l := &Label{
		Label:   *label,
		Current: content,
	}

	// reusable primitive
	l.r = NewReusable(func(nl *nullLabel) {
		l.SetLabelUnsafe(nl.Rich)
	})

	return l
}

// NewInheritLabel creates a new label wrapper for structs that inherit this
// label.
func NewInheritLabel(super SuperLabeler) *Label {
	l := NewLabel(text.Rich{})
	l.super = super
	return l
}

func (l *Label) validsuper() bool {
	_, ok := l.super.(*Label)
	// supers must not be the current struct and must not be nil.
	return !ok && l.super != nil
}

// Reset wipes the state to be just after construction. If super is not nil,
// then it's reset as well.
func (l *Label) Reset() {
	l.Current = text.Rich{}
	l.r.Invalidate()
	l.Label.SetText("")

	if l.validsuper() {
		l.super.Reset()
	}
}

func (l *Label) AsyncSetLabel(fn LabelerFn, info string) {
	AsyncUse(l.r, func(ctx context.Context) (interface{}, func(), error) {
		nl := &nullLabel{}
		f, err := fn(ctx, nl)
		return nl, f, err
	})
}

// SetLabel is thread-safe.
func (l *Label) SetLabel(content text.Rich) {
	gts.ExecAsync(func() { l.SetLabelUnsafe(content) })
}

// SetLabelUnsafe sets the label in the current thread, meaning it's not
// thread-safe. If this label has a super, then it will call that struct's
// SetLabelUnsafe instead of its own.
func (l *Label) SetLabelUnsafe(content text.Rich) {
	l.Current = content

	if l.validsuper() {
		l.super.SetLabelUnsafe(content)
	} else {
		l.SetMarkup(markup.Render(content))
	}
}

// GetLabel is NOT thread-safe.
func (l *Label) GetLabel() text.Rich {
	return l.Current
}

// GetText is NOT thread-safe.
func (l *Label) GetText() string {
	return l.Current.Content
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
