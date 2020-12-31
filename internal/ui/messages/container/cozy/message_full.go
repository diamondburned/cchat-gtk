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

	// Header wraps author and timestamp.
	HeaderBox *gtk.Box
}

type AvatarPixbufCopier interface {
	CopyAvatarPixbuf(img httputil.SurfaceContainer)
}

var (
	_ AvatarPixbufCopier    = (*FullMessage)(nil)
	_ message.Container     = (*FullMessage)(nil)
	_ container.GridMessage = (*FullMessage)(nil)
)

var boldCSS = primitives.PrepareCSS(`
	* { font-weight: 600; }
`)

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
	avatar := NewAvatar()
	avatar.SetMarginTop(TopFullMargin)
	avatar.SetMarginStart(container.ColumnSpacing * 2)
	avatar.Connect("clicked", func(w gtk.IWidget) {
		if output := gc.Username.Output(); len(output.Mentions) > 0 {
			labeluri.PopoverMentioner(w, output.Input, output.Mentions[0])
		}
	})
	// We don't call avatar.Show(). That's called in Attach.

	// Style the timestamp accordingly.
	gc.Timestamp.SetXAlign(0.0)           // left-align
	gc.Timestamp.SetVAlign(gtk.ALIGN_END) // bottom-align
	gc.Timestamp.SetMarginStart(0)        // clear margins

	gc.Username.SetMaxWidthChars(75)

	// Attach the class and CSS for the left avatar.
	avatarCSS(avatar)

	// Attach the username style provider.
	primitives.AttachCSS(gc.Username, boldCSS)

	header, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	header.PackStart(gc.Username, false, false, 0)
	header.PackStart(gc.Timestamp, false, false, 7) // padding
	header.Show()

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.PackStart(header, false, false, 0)
	main.PackStart(gc.Content, false, false, 0)
	main.SetMarginTop(TopFullMargin)
	main.SetMarginEnd(container.ColumnSpacing * 2)
	main.Show()

	// Also attach a class for the main box shown on the right.
	primitives.AddClass(main, "cozy-main")

	return &FullMessage{
		GenericContainer: gc,
		Avatar:           avatar,
		MainBox:          main,
		HeaderBox:        header,
	}
}

func (m *FullMessage) Collapsed() bool { return false }

func (m *FullMessage) Unwrap(grid *gtk.Grid) *message.GenericContainer {
	// Remove GenericContainer's widgets from the containers.
	m.HeaderBox.Remove(m.Username)
	m.HeaderBox.Remove(m.Timestamp)
	m.MainBox.Remove(m.HeaderBox)
	m.MainBox.Remove(m.Content)

	// Hide the avatar.
	m.Avatar.Hide()

	// Remove the message from the grid.
	grid.Remove(m.Avatar)
	grid.Remove(m.MainBox)

	// Return after removing.
	return m.GenericContainer
}

func (m *FullMessage) Attach() []gtk.IWidget {
	m.Avatar.Show()
	return []gtk.IWidget{m.Avatar, m.MainBox}
}

func (m *FullMessage) Focusable() gtk.IWidget {
	return m.Avatar
}

func (m *FullMessage) UpdateTimestamp(t time.Time) {
	m.GenericContainer.UpdateTimestamp(t)
	m.Timestamp.SetText(humanize.TimeAgoLong(t))
}

func (m *FullMessage) UpdateAuthor(author cchat.Author) {
	// Call the parent's method to update the labels.
	m.GenericContainer.UpdateAuthor(author)
	m.Avatar.SetURL(author.Avatar())
}

// CopyAvatarPixbuf sets the pixbuf into the given container. This shares the
// same pixbuf, but gtk.Image should take its own reference from the pixbuf.
func (m *FullMessage) CopyAvatarPixbuf(dst httputil.SurfaceContainer) {
	switch img := m.Avatar.Image.GetImage(); img.GetStorageType() {
	case gtk.IMAGE_PIXBUF:
		dst.SetFromPixbuf(img.GetPixbuf())
	case gtk.IMAGE_ANIMATION:
		dst.SetFromAnimation(img.GetAnimation())
	case gtk.IMAGE_SURFACE:
		v, _ := img.GetProperty("surface")
		dst.SetFromSurface(v.(*cairo.Surface))
	}
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
	// _ AvatarPixbufCopier    = (*FullSendingMessage)(nil)
	_ message.Container     = (*FullSendingMessage)(nil)
	_ container.GridMessage = (*FullSendingMessage)(nil)
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
