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
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/menu"
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
	CopyAvatarPixbuf(img httputil.ImageContainer)
}

var (
	_ AvatarPixbufCopier    = (*FullMessage)(nil)
	_ message.Container     = (*FullMessage)(nil)
	_ container.GridMessage = (*FullMessage)(nil)
)

var boldCSS = primitives.PrepareCSS(`
	* { font-weight: 600; }
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
	// We don't call avatar.Show(). That's called in Attach.

	// Style the timestamp accordingly.
	gc.Timestamp.SetXAlign(0.0)           // left-align
	gc.Timestamp.SetVAlign(gtk.ALIGN_END) // bottom-align
	gc.Timestamp.SetMarginStart(0)        // clear margins

	// Attach the class for the left avatar.
	primitives.AddClass(avatar, "cozy-avatar")

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
	m.MainBox.Remove(m.Content)

	// Return after removing.
	return m.GenericContainer
}

func (m *FullMessage) UpdateTimestamp(t time.Time) {
	m.GenericContainer.UpdateTimestamp(t)
	m.Timestamp.SetText(humanize.TimeAgoLong(t))
}

func (m *FullMessage) UpdateAuthor(author cchat.MessageAuthor) {
	// Call the parent's method to update the labels.
	m.GenericContainer.UpdateAuthor(author)

	// If the author has an avatar:
	if avatarer, ok := author.(cchat.MessageAuthorAvatar); ok {
		m.Avatar.SetURL(avatarer.Avatar())
	}
}

// CopyAvatarPixbuf sets the pixbuf into the given container. This shares the
// same pixbuf, but gtk.Image should take its own reference from the pixbuf.
func (m *FullMessage) CopyAvatarPixbuf(dst httputil.ImageContainer) {
	switch m.Avatar.GetStorageType() {
	case gtk.IMAGE_PIXBUF:
		dst.SetFromPixbuf(m.Avatar.GetPixbuf())
	case gtk.IMAGE_ANIMATION:
		dst.SetFromAnimation(m.Avatar.GetAnimation())
	}
}

func (m *FullMessage) Attach(grid *gtk.Grid, row int) {
	m.Avatar.Show()
	container.AttachRow(grid, row, m.Avatar, m.MainBox)
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
	roundimage.Image
	url string
}

func NewAvatar() *Avatar {
	avatar, _ := roundimage.NewImage(0)
	avatar.SetSizeRequest(AvatarSize, AvatarSize)
	avatar.SetVAlign(gtk.ALIGN_START)

	// Default icon.
	primitives.SetImageIcon(avatar.Image, "user-available-symbolic", AvatarSize)

	return &Avatar{*avatar, ""}
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
	httputil.AsyncImageSized(a, url, AvatarSize, AvatarSize)
}

// ManuallySetURL sets the URL without downloading the image. It assumes the
// pixbuf is borrowed elsewhere.
func (a *Avatar) ManuallySetURL(url string) {
	a.url = url
}
