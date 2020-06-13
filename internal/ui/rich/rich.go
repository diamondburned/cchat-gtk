package rich

import (
	"html"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func Small(text string) string {
	return `<span size="small" color="#808080">` + text + "</span>"
}

func MakeRed(content text.Rich) string {
	return `<span color="red">` + html.EscapeString(content.Content) + `</span>`
}

type Labeler interface {
	// thread-safe
	cchat.LabelContainer // thread-safe

	// not thread-safe
	SetLabelUnsafe(text.Rich)
	GetLabel() text.Rich
	GetText() string
}

type Label struct {
	gtk.Label
	current text.Rich
}

var (
	_ gtk.IWidget = (*Label)(nil)
	_ Labeler     = (*Label)(nil)
)

func NewLabel(content text.Rich) *Label {
	label, _ := gtk.LabelNew("")
	label.SetMarkup(parser.RenderMarkup(content))
	label.SetHAlign(gtk.ALIGN_START)
	return &Label{*label, content}
}

// Reset wipes the state to be just after construction.
func (l *Label) Reset() {
	l.current = text.Rich{}
	l.Label.SetText("")
}

func (l *Label) AsyncSetLabel(fn func(cchat.LabelContainer) error, info string) {
	go func() {
		if err := fn(l); err != nil {
			log.Error(errors.Wrap(err, info))
		}
	}()
}

// SetLabel is thread-safe.
func (l *Label) SetLabel(content text.Rich) {
	gts.ExecAsync(func() {
		l.SetLabelUnsafe(content)
	})
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
