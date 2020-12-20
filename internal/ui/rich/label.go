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
	"github.com/pkg/errors"
)

type Labeler interface {
	// thread-safe
	cchat.LabelContainer // thread-safe

	// not thread-safe
	SetLabelUnsafe(text.Rich)
	GetLabel() text.Rich
	GetText() string
}

// SuperLabeler represents a label that inherits the current labeler.
type SuperLabeler interface {
	SetLabelUnsafe(text.Rich)
}

type LabelerFn = func(context.Context, cchat.LabelContainer) (func(), error)

type Label struct {
	gtk.Label
	Current text.Rich

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

func (l *Label) AsyncSetLabel(fn LabelerFn, info string) {
	ctx := primitives.HandleDestroyCtx(context.Background(), l)
	gts.Async(func() (func(), error) {
		f, err := fn(ctx, l)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load iconer")
		}

		return func() { l.Connect("destroy", f) }, nil
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
