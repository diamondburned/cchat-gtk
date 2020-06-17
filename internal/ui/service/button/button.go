package button

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
	"github.com/diamondburned/cchat/text"
)

type ToggleButtonImage struct {
	rich.ToggleButtonImage

	extraMenu []menu.Item
	menu      *menu.LazyMenu

	clicked func(bool)

	err    error
	icon   bool // whether or not the button has an icon
	iconSz int
}

var _ cchat.IconContainer = (*ToggleButtonImage)(nil)

func NewToggleButtonImage(content text.Rich) *ToggleButtonImage {
	b := rich.NewToggleButtonImage(content)
	b.Show()

	tb := &ToggleButtonImage{
		ToggleButtonImage: *b,

		clicked: func(bool) {},
		menu:    menu.NewLazyMenu(b.ToggleButton),
	}

	tb.Connect("clicked", func() {
		tb.clicked(tb.GetActive())
	})

	return tb
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

	if b.icon {
		b.Image.SetPlaceholderIcon("user-available-symbolic", b.Image.Size())
	}
}

func (b *ToggleButtonImage) SetLoading() {
	b.SetLabelUnsafe(b.GetLabel())

	// Reset the menu.
	b.menu.SetItems(b.extraMenu)

	if b.icon {
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
	if b.icon {
		b.Image.SetPlaceholderIcon("computer-fail-symbolic", b.Image.Size())
	}
}

func (b *ToggleButtonImage) SetPlaceholderIcon(iconName string, iconSzPx int) {
	b.icon = true
	b.Image.SetPlaceholderIcon(iconName, iconSzPx)
}

func (b *ToggleButtonImage) SetIcon(url string) {
	gts.ExecAsync(func() { b.SetIconUnsafe(url) })
}

func (b *ToggleButtonImage) SetIconUnsafe(url string) {
	b.icon = true
	b.Image.SetIconUnsafe(url)
}

// type Row struct {
// 	gtk.Box
// 	Button   *ToggleButtonImage
// 	Children *gtk.Box
// }
