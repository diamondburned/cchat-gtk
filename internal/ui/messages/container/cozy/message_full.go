package cozy

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts/httputil"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

type FullMessage struct {
	*message.GenericContainer

	// Grid widgets.
	Avatar  *gtk.Image
	MainBox *gtk.Box // wraps header and content

	// Header wraps author and timestamp.
	HeaderBox *gtk.Box
}

var (
	_ AvatarPixbufCopier    = (*FullMessage)(nil)
	_ message.Container     = (*FullMessage)(nil)
	_ container.GridMessage = (*FullMessage)(nil)
)

func NewFullMessage(msg cchat.MessageCreate) *FullMessage {
	msgc := WrapFullMessage(message.NewContainer(msg))
	// Don't update the avatar.
	msgc.UpdateContent(msg.Content())
	msgc.UpdateAuthorName(msg.Author().Name())
	msgc.UpdateTimestamp(msg.Time())
	return msgc
}

func WrapFullMessage(gc *message.GenericContainer) *FullMessage {
	avatar, _ := gtk.ImageNew()
	avatar.SetSizeRequest(AvatarSize, AvatarSize)
	avatar.SetVAlign(gtk.ALIGN_START)
	avatar.SetMarginStart(container.ColumnSpacing * 2)
	avatar.Show()

	header, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	header.PackStart(gc.Username, false, false, 0)
	header.PackStart(gc.Timestamp, false, false, 7) // padding
	header.Show()

	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	main.PackStart(header, false, false, 0)
	main.PackStart(gc.Content, false, false, 2)
	main.SetMarginBottom(2)
	main.SetMarginEnd(container.ColumnSpacing * 2)
	main.Show()

	return &FullMessage{
		GenericContainer: gc,
		Avatar:           avatar,
		MainBox:          main,
		HeaderBox:        header,
	}
}

func (m *FullMessage) UpdateAuthor(author cchat.MessageAuthor) {
	// Call the parent's method to update the labels.
	m.GenericContainer.UpdateAuthor(author)
	m.updateAuthorAvatar(author)
}

func (m *FullMessage) updateAuthorAvatar(author cchat.MessageAuthor) {
	// If the author has an avatar:
	if avatarer, ok := author.(cchat.MessageAuthorAvatar); ok {
		// Download the avatar asynchronously.
		httputil.AsyncImageSized(
			m.Avatar,
			avatarer.Avatar(),
			AvatarSize, AvatarSize,
			imgutil.Round(true),
		)
	}
}

func (m *FullMessage) CopyAvatarPixbuf(dst httputil.ImageContainer) {
	switch m.Avatar.GetStorageType() {
	case gtk.IMAGE_PIXBUF:
		dst.SetFromPixbuf(m.Avatar.GetPixbuf())
	case gtk.IMAGE_ANIMATION:
		dst.SetFromAnimation(m.Avatar.GetAnimation())
	}
}

func (m *FullMessage) Attach(grid *gtk.Grid, row int) {
	container.AttachRow(grid, row, m.Avatar, m.MainBox)
}

type FullSendingMessage struct {
	message.PresendContainer
	FullMessage
}

var (
	_ AvatarPixbufCopier    = (*FullSendingMessage)(nil)
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
