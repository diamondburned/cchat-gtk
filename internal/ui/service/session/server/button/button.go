package button

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
)

const UnreadColorDefs = `
	@define-color mentioned rgb(240, 71, 71);
`

type ToggleButtonImage struct {
	*rich.ToggleButtonImage

	extraMenu []menu.Item
	menu      *menu.LazyMenu

	clicked func(bool)
	readcss primitives.ClassEnum

	err    error
	icon   string // whether or not the button has an icon
	iconSz int
}

var _ cchat.IconContainer = (*ToggleButtonImage)(nil)

var serverButtonCSS = primitives.PrepareClassCSS("server-button", `
	.selected-server {
		border-left: 2px solid mix(@theme_base_color, @theme_fg_color, 0.1);
		background-color:      mix(@theme_base_color, @theme_fg_color, 0.1);
		color: @theme_fg_color;
	}

	.read {
		color: alpha(@theme_fg_color, 0.5);
		border-left: 2px solid transparent;
	}

	.unread {
		color: @theme_fg_color;
		border-left: 2px solid alpha(@theme_fg_color, 0.75);
		background-color: alpha(@theme_fg_color, 0.05);
	}

	.mentioned {
		color: @mentioned;
		border-left: 2px solid alpha(@mentioned, 0.75);
		background-color: alpha(@mentioned, 0.05);
	}

`+UnreadColorDefs)

func NewToggleButtonImage(content text.Rich) *ToggleButtonImage {
	b := rich.NewToggleButtonImage(content)
	return WrapToggleButtonImage(b)
}

func WrapToggleButtonImage(b *rich.ToggleButtonImage) *ToggleButtonImage {
	b.Show()

	tb := &ToggleButtonImage{
		ToggleButtonImage: b,

		clicked: func(bool) {},
		menu:    menu.NewLazyMenu(b.ToggleButton),
	}
	tb.Connect("clicked", func() { tb.clicked(tb.GetActive()) })
	serverButtonCSS(tb)

	return tb
}

func (b *ToggleButtonImage) SetSelected(selected bool) {
	// Set the clickability the opposite as the boolean.
	// b.SetSensitive(!selected)
	b.SetInconsistent(selected)

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

func (b *ToggleButtonImage) SetClicked(clicked func(bool)) {
	b.clicked = clicked
}

func (b *ToggleButtonImage) SetClickedIfTrue(clickedIfTrue func()) {
	b.clicked = func(clicked bool) {
		if clicked {
			clickedIfTrue()
		}
	}
}

func (b *ToggleButtonImage) SetNormalExtraMenu(items []menu.Item) {
	b.extraMenu = items
	b.SetNormal()
}

func (b *ToggleButtonImage) SetNormal() {
	b.SetLabelUnsafe(b.GetLabel())
	b.menu.SetItems(b.extraMenu)

	if b.icon != "" {
		b.Image.SetPlaceholderIcon(b.icon, b.Image.Size())
	}
}

func (b *ToggleButtonImage) SetLoading() {
	b.SetLabelUnsafe(b.GetLabel())

	// Reset the menu.
	b.menu.SetItems(b.extraMenu)

	if b.icon != "" {
		b.Image.SetPlaceholderIcon("content-loading-symbolic", b.Image.Size())
	}
}

func (b *ToggleButtonImage) SetFailed(err error, retry func()) {
	b.Label.SetMarkup(rich.MakeRed(b.GetLabel()))

	// Add a retry button, if any.
	b.menu.Reset()
	b.menu.AddItems(menu.SimpleItem("Retry", retry))
	b.menu.AddItems(b.extraMenu...)

	// If we have an icon set, then we can use the failed icon.
	if b.icon != "" {
		b.Image.SetPlaceholderIcon("computer-fail-symbolic", b.Image.Size())
	}
}

func (b *ToggleButtonImage) SetUnreadUnsafe(unread, mentioned bool) {
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

func (b *ToggleButtonImage) SetPlaceholderIcon(iconName string, iconSzPx int) {
	b.icon = iconName
	b.Image.SetPlaceholderIcon(iconName, iconSzPx)
}

func (b *ToggleButtonImage) SetIcon(url string) {
	gts.ExecAsync(func() { b.SetIconUnsafe(url) })
}

func (b *ToggleButtonImage) SetIconUnsafe(url string) {
	b.icon = ""
	b.Image.SetIconUnsafe(url)
}
