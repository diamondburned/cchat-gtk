package cozy

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/menu"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gtk"
)

// TopFullMargin is the margin on top of every full message.
const TopFullMargin = 4

type FullMessage struct {
	*message.GenericContainer

	// Grid widgets.
	Avatar  *Avatar
	MainBox *gtk.Box // wraps header and content

	Header    *labeluri.Label
	timestamp string // markup
}

type AvatarPixbufCopier interface {
	CopyAvatarPixbuf(img httputil.SurfaceContainer) bool
}

var (
	_ AvatarPixbufCopier   = (*FullMessage)(nil)
	_ message.Container    = (*FullMessage)(nil)
	_ container.MessageRow = (*FullMessage)(nil)
)

var avatarCSS = primitives.PrepareClassCSS("cozy-avatar", `
	/* Slightly dip down on click */
	.cozy-avatar:active {
	    margin-top: 1px;
	}
`)

func NewFullMessage(msg cchat.MessageCreate) *FullMessage {
	msgc := WrapFullMessage(message.NewContainer(msg))
	// Don't update the avatar. NewMessage in controller will try and reuse the
	// pixbuf if possible.
	msgc.UpdateAuthorName(msg.Author().Name())
	msgc.UpdateTimestamp(msg.Time())
	msgc.UpdateContent(msg.Content(), false)
	return msgc
}

func WrapFullMessage(gc *message.GenericContainer) *FullMessage {
	header := labeluri.NewLabel(text.Rich{})
	header.SetHAlign(gtk.ALIGN_START) // left-align
	header.SetMaxWidthChars(100)
	header.Show()

	avatar := NewAvatar()
	avatar.SetMarginTop(TopFullMargin / 2)
	avatar.SetMarginStart(container.ColumnSpacing * 2)
	avatar.Connect("clicked", func(w gtk.IWidget) {
		if output := header.Output(); len(output.Mentions) > 0 {
			labeluri.PopoverMentioner(w, output.Input, output.Mentions[0])
		}
	})
	avatar.Show()

	// Attach the class and CSS for the left avatar.
	avatarCSS(avatar)

	// Attach the username style provider.
	// primitives.AttachCSS(gc.Username, boldCSS)

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.PackStart(header, false, false, 0)
	main.PackStart(gc.Content, false, false, 0)
	main.SetMarginTop(TopFullMargin)
	main.SetMarginEnd(container.ColumnSpacing * 2)
	main.SetMarginStart(container.ColumnSpacing)
	main.Show()

	// Also attach a class for the main box shown on the right.
	primitives.AddClass(main, "cozy-main")

	gc.PackStart(avatar, false, false, 0)
	gc.PackStart(main, true, true, 0)
	gc.SetClass("cozy-full")

	return &FullMessage{
		GenericContainer: gc,

		Avatar:  avatar,
		MainBox: main,
		Header:  header,
	}
}

func (m *FullMessage) Collapsed() bool { return false }

func (m *FullMessage) Unwrap() *message.GenericContainer {
	// Remove GenericContainer's widgets from the containers.
	m.Header.Destroy()
	m.MainBox.Remove(m.Content) // not ours, so don't destroy.

	// Remove the message from the grid.
	m.Avatar.Destroy()
	m.MainBox.Destroy()

	// Return after removing.
	return m.GenericContainer
}

func (m *FullMessage) UpdateTimestamp(t time.Time) {
	m.GenericContainer.UpdateTimestamp(t)

	m.timestamp = "  " +
		`<span alpha="70%" size="small">` + humanize.TimeAgoLong(t) + `</span>`

	// Update the timestamp.
	m.Header.SetMarkup(m.Header.Output().Markup + m.timestamp)
}

func (m *FullMessage) UpdateAuthor(author cchat.Author) {
	// Call the parent's method to update the state.
	m.GenericContainer.UpdateAuthor(author)
	m.UpdateAuthorName(author.Name())
	m.Avatar.SetURL(author.Avatar())
}

func (m *FullMessage) UpdateAuthorName(name text.Rich) {
	cfg := markup.RenderConfig{}
	cfg.NoReferencing = true
	cfg.SetForegroundAnchor(m.ContentBodyStyle)

	output := markup.RenderCmplxWithConfig(name, cfg)
	output.Markup = `<span font_weight="600">` + output.Markup + "</span>"

	m.Header.SetMarkup(output.Markup + m.timestamp)
	m.Header.SetUnderlyingOutput(output)
}

// CopyAvatarPixbuf sets the pixbuf into the given container. This shares the
// same pixbuf, but gtk.Image should take its own reference from the pixbuf.
func (m *FullMessage) CopyAvatarPixbuf(dst httputil.SurfaceContainer) bool {
	switch img := m.Avatar.Image.GetImage(); img.GetStorageType() {
	case gtk.IMAGE_PIXBUF:
		dst.SetFromPixbuf(img.GetPixbuf())
	case gtk.IMAGE_ANIMATION:
		dst.SetFromAnimation(img.GetAnimation())
	case gtk.IMAGE_SURFACE:
		v, _ := img.GetProperty("surface")
		dst.SetFromSurface(v.(*cairo.Surface))
	default:
		return false
	}
	return true
}

func (m *FullMessage) AttachMenu(items []menu.Item) {
	// Bind to parent's container as well.
	m.GenericContainer.AttachMenu(items)

	// Bind to the box.
	// TODO lol
}

type FullSendingMessage struct {
	message.PresendContainer
	FullMessage
}

var (
	_ message.Container    = (*FullSendingMessage)(nil)
	_ container.MessageRow = (*FullSendingMessage)(nil)
)

func NewFullSendingMessage(msg input.PresendMessage) *FullSendingMessage {
	var msgc = message.NewPresendContainer(msg)

	return &FullSendingMessage{
		PresendContainer: msgc,
		FullMessage:      *WrapFullMessage(msgc.GenericContainer),
	}
}

type Avatar struct {
	roundimage.Button
	Image *roundimage.StaticImage
	url   string
}

func NewAvatar() *Avatar {
	img, _ := roundimage.NewStaticImage(nil, 0)
	img.SetSizeRequest(AvatarSize, AvatarSize)
	img.Show()

	avatar, _ := roundimage.NewCustomButton(img)
	avatar.SetVAlign(gtk.ALIGN_START)

	// Default icon.
	primitives.SetImageIcon(img, "user-available-symbolic", AvatarSize)

	return &Avatar{*avatar, img, ""}
}

// SetURL updates the Avatar to be that URL. It does nothing if URL is empty or
// matches the existing one.
func (a *Avatar) SetURL(url string) {
	// Check if the URL is the same. This will save us quite a few requests, as
	// some methods rely on the side-effects of other methods, and they may call
	// UpdateAuthor multiple times.
	if a.url == url || url == "" {
		return
	}

	a.url = url
	a.Image.SetImageURL(url)
}

// ManuallySetURL sets the URL without downloading the image. It assumes the
// pixbuf is borrowed elsewhere.
func (a *Avatar) ManuallySetURL(url string) {
	a.url = url
}
