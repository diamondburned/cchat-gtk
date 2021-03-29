package button

import (
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/gotk3/gotk3/gtk"
)

const UnreadColorDefs = `
	@define-color mentioned rgb(240, 71, 71);
`

const IconSize = 38

type ToggleButton struct {
	gtk.ToggleButton
	Label    *rich.Label
	labelRev *gtk.Revealer

	// These fields are nil if image is false.
	Image    *roundimage.StillImage
	imageRev *gtk.Revealer

	Box *gtk.Box

	state   rich.LabelStateStorer
	clicked func(bool)
	readcss primitives.ClassEnum

	icon  string // whether or not the button has an icon
	label bool
}

var serverButtonCSS = primitives.PrepareClassCSS("server-button", `
	.server-button {
		min-width: 0px;
	}

	.selected-server {
		border-left: 2px solid mix(@theme_base_color, @theme_fg_color, 0.1);
		background-color:      mix(@theme_base_color, @theme_fg_color, 0.1);
		color: @theme_fg_color;
	}

	.read {
		/* color: alpha(@theme_fg_color, 0.5); */
		border-left: 2px solid transparent;
	}

	.unread {
		color: @theme_fg_color;
		border-left: 2px solid alpha(@theme_fg_color, 0.75);
		/* background-color: alpha(@theme_fg_color, 0.05); */
	}

	.mentioned {
		color: @mentioned;
		border-left: 2px solid alpha(@mentioned, 0.75);
		background-color: alpha(@mentioned, 0.05);
	}

`+UnreadColorDefs)

// NewToggleButton creates a new toggle button.
func NewToggleButton(state rich.LabelStateStorer) *ToggleButton {
	label := rich.NewLabelWithRenderer(state, rich.RenderSkipImages)
	label.SetMarginStart(5)
	label.Show()

	labelRev, _ := gtk.RevealerNew()
	labelRev.Add(label)
	labelRev.SetRevealChild(true)
	labelRev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	labelRev.SetTransitionDuration(100)
	labelRev.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.SetHAlign(gtk.ALIGN_START)
	box.PackStart(labelRev, false, false, 0)
	box.Show()

	button, _ := gtk.ToggleButtonNew()
	button.SetRelief(gtk.RELIEF_NONE)
	button.Add(box)
	button.Show()

	tb := &ToggleButton{
		ToggleButton: *button,
		Label:        label,
		labelRev:     labelRev,
		Box:          box,
		state:        state,
		clicked:      func(bool) {},
	}

	tb.SetShowLabel(true)
	tb.Connect("clicked", func(w *gtk.ToggleButton) { tb.clicked(w.GetActive()) })
	serverButtonCSS(tb)

	// Ensure that we display an icon when we receive one.
	state.OnUpdate(func() {
		if state.Image().HasImage() {
			tb.ensureImage()
		}
	})

	return tb
}

// SetSelected sets the button's intermediate state and appearance to look
// like it's clicked without triggering the callback.
func (b *ToggleButton) SetSelected(selected bool) {
	if selected {
		primitives.AddClass(b, "selected-server")
	} else {
		primitives.RemoveClass(b, "selected-server")
	}

	// Some special edge case that I forgot.
	if !selected {
		b.SetActive(false)
	}
}

// SetShowLabel sets whether or not to show the button's label. If the button
// does not have an image, then the label is always shown.
func (b *ToggleButton) SetShowLabel(showLabel bool) {
	b.label = showLabel

	// Enforce particular rules that are unfeasible at the moment. When these
	// conditions change elsewhere, this function will be called again.
	if b.imageRev != nil {
		showLabel = showLabel || !b.imageRev.GetRevealChild()
	} else {
		showLabel = true
	}

	// Expand the box when we're showing the label.
	b.SetHExpand(showLabel)
	b.labelRev.SetRevealChild(showLabel)
}

// GetShowLabel gets whether or not the label is being shown.
func (b *ToggleButton) GetShowLabel() bool {
	return b.labelRev.GetRevealChild() || b.label
}

// SetClicked sets the callback to run when clicked. It overrides the previous
// callback.
func (b *ToggleButton) SetClicked(clicked func(bool)) {
	b.clicked = clicked
}

func (b *ToggleButton) SetClickedIfTrue(clickedIfTrue func()) {
	b.clicked = func(clicked bool) {
		if clicked {
			clickedIfTrue()
		}
	}
}

func (b *ToggleButton) ensureImage() {
	if b.Image != nil {
		return
	}

	b.Image = roundimage.NewStillImage(b, 0)
	b.Image.SetSizeRequest(IconSize, IconSize)
	b.Image.Show()

	// TODO: tooltip false once hover is implemented.
	rich.BindRoundImage(b.Image, b.state, true)

	b.imageRev, _ = gtk.RevealerNew()
	b.imageRev.Add(b.Image)
	b.imageRev.SetRevealChild(true)
	b.imageRev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_RIGHT)
	b.imageRev.SetTransitionDuration(75)
	b.imageRev.Show()

	b.Box.PackStart(b.imageRev, false, false, 0)
	b.Box.ReorderChild(b.imageRev, 0)

	// Set the callback to render the user's initials if there is no name.
	b.Image.UseInitialsIfNone(func() string {
		return b.state.Label().String()
	})

	// Restore the label's visible state now that we have an image.
	b.SetShowLabel(b.label)
}

// UseEmptyIcon forces the ToggleButton to show an icon, even if it's a
// placeholder.
func (b *ToggleButton) UseEmptyIcon() {
	b.ensureImage()
}

// SetNormal sets the button's state to normal from either loading or failed.
func (b *ToggleButton) SetNormal() {
	b.Label.SetRenderer(rich.RenderSkipImages)
	b.SetSensitive(true)

	if b.Image != nil && b.icon != "" {
		b.SetPlaceholderIcon(b.icon, b.Image.Size())
	}
}

// SetLoading sets the button's state to loading.
func (b *ToggleButton) SetLoading() {
	b.Label.SetRenderer(rich.RenderSkipImages)
	b.SetSensitive(false)

	if b.Image != nil && b.icon != "" {
		b.SetPlaceholderIcon("content-loading-symbolic", b.Image.Size())
	}
}

func (b *ToggleButton) SetFailed(err error, retry func()) {
	b.Label.SetRenderer(rich.MakeRed)
	b.SetSensitive(true)

	// If we have an icon set, then we can use the failed icon.
	if b.Image != nil && b.icon != "" {
		b.SetPlaceholderIcon("computer-fail-symbolic", b.Image.Size())
	}
}

func (b *ToggleButton) SetUnreadUnsafe(unread, mentioned bool) {
	switch {
	// Prioritize mentions over unreads.
	case mentioned:
		b.readcss.SetClass(b, "mentioned")
	case unread:
		b.readcss.SetClass(b, "unread")
	default:
		b.readcss.SetClass(b, "read")
	}
}

func (b *ToggleButton) SetPlaceholderIcon(iconName string, iconSzPx int) {
	b.icon = iconName
	b.ensureImage()
	b.Image.SetPlaceholderIcon(iconName, iconSzPx)
}
