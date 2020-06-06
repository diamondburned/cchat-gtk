package rich

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

// TODO: parser

type Labeler interface {
	cchat.LabelContainer // thread-safe
	GetLabel() text.Rich // not thread-safe
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

type ToggleButtonImage struct {
	gtk.ToggleButton
	Labeler

	Label gtk.Label
	Image gtk.Image

	Box gtk.Box
}

var (
	_ gtk.IWidget          = (*ToggleButton)(nil)
	_ cchat.LabelContainer = (*ToggleButton)(nil)
)

func NewToggleButtonImage(content text.Rich, iconName string) *ToggleButtonImage {
	l := NewLabel(content)
	l.Show()

	var i *gtk.Image
	if iconName != "" {
		i, _ = gtk.ImageNewFromIconName(iconName, gtk.ICON_SIZE_BUTTON)
	} else {
		i, _ = gtk.ImageNew()
	}
	i.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(i, false, false, 0)
	box.PackStart(l, true, true, 5)
	box.Show()

	b, _ := gtk.ToggleButtonNew()
	b.Add(box)

	return &ToggleButtonImage{
		ToggleButton: *b,
		Labeler:      l, // easy inheritance of methods
		Label:        l.Label,
		Image:        *i,
		Box:          *box,
	}
}
