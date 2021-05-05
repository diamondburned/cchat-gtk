package cozy

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

type FullMessage struct {
	*message.State

	// Grid widgets.
	Avatar  *Avatar
	MainBox *gtk.Box // wraps header and content

	HeaderLabel *labeluri.Label
	timestamp   string // markup

	// unwrap is used to removing label handlers.
	unwrap func()
}

var (
	_ message.Container    = (*FullMessage)(nil)
	_ container.MessageRow = (*FullMessage)(nil)
)

var avatarCSS = primitives.PrepareClassCSS("cozy-avatar", `
	.cozy-avatar {
		margin-top: 2px;
	}

	/* Slightly dip down on click */
	.cozy-avatar:active {
	    margin-top: 1px;
	}
`)

var mainCSS = primitives.PrepareClassCSS("cozy-main", `
	.cozy-main {
		margin-top: 4px;
	}
`)

func NewFullMessage(msg cchat.MessageCreate) *FullMessage {
	return WrapFullMessage(message.NewState(msg))
}

func WrapFullMessage(gc *message.State) *FullMessage {
	header := labeluri.NewLabel(text.Rich{})
	header.SetHAlign(gtk.ALIGN_START) // left-align
	header.SetMaxWidthChars(100)
	header.Show()

	avatar := NewAvatar(gc.Row)
	avatar.SetMarginStart(container.ColumnSpacing)
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
	main.SetMarginEnd(container.ColumnSpacing)
	main.SetMarginStart(container.ColumnSpacing)
	main.Show()
	mainCSS(main)

	gc.PackStart(avatar, false, false, 0)
	gc.PackStart(main, true, true, 0)
	gc.SetClass("cozy-full")

	removeUpdate := gc.Author.Name.OnUpdate(func() {
		avatar.SetImage(gc.Author.Name.Image())
		header.SetLabel(gc.Author.Name.Label())
	})

	msg := &FullMessage{
		State:     gc,
		timestamp: formatLongTime(gc.Time),

		Avatar:      avatar,
		MainBox:     main,
		HeaderLabel: header,

		unwrap: func() { removeUpdate() },
	}

	cfg := markup.RenderConfig{}
	cfg.NoReferencing = true
	cfg.SetForegroundAnchor(gc.ContentBodyStyle)

	header.SetRenderer(func(rich text.Rich) markup.RenderOutput {
		output := markup.RenderCmplxWithConfig(rich, cfg)
		output.Markup = `<span font_weight="600">` + output.Markup + "</span>"
		output.Markup += msg.timestamp

		return output
	})

	return msg
}

func (m *FullMessage) Revert() *message.State {
	// Remove the handlers.
	m.unwrap()

	// Destroy the bottom leaf widgets first.
	m.Avatar.Destroy()
	m.HeaderLabel.Destroy()

	// Remove the content label from main then destroy it, in case destroying it
	// ruins the label.
	m.MainBox.Remove(m.Content)
	m.MainBox.Destroy()

	m.ClearBox()

	return m.Unwrap()
}

type full interface{ full() }

func (m *FullMessage) full() {}

func formatLongTime(t time.Time) string {
	return ` <span alpha="70%" size="small">` + humanize.TimeAgoLong(t) + `</span>`
}

type FullSendingMessage struct {
	*FullMessage
	message.Presender
}

var (
	_ message.Container    = (*FullSendingMessage)(nil)
	_ container.MessageRow = (*FullSendingMessage)(nil)
)

func WrapFullSendingMessage(pstate *message.PresendState) *FullSendingMessage {
	return &FullSendingMessage{
		FullMessage: WrapFullMessage(pstate.State),
		Presender:   pstate,
	}
}
