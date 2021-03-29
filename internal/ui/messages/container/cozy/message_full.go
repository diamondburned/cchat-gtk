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

// TopFullMargin is the margin on top of every full message.
const TopFullMargin = 4

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
	/* Slightly dip down on click */
	.cozy-avatar:active {
	    margin-top: 1px;
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

	msg := &FullMessage{
		State:     gc,
		timestamp: formatLongTime(gc.Time),

		Avatar:      avatar,
		MainBox:     main,
		HeaderLabel: header,

		unwrap: gc.Author.Name.OnUpdate(func() {
			avatar.SetImage(gc.Author.Name.Image())
			header.SetLabel(gc.Author.Name.Label())
		}),
	}

	header.SetRenderer(func(rich text.Rich) markup.RenderOutput {
		cfg := markup.RenderConfig{}
		cfg.NoReferencing = true
		cfg.SetForegroundAnchor(gc.ContentBodyStyle)

		output := markup.RenderCmplxWithConfig(rich, cfg)
		output.Markup = `<span font_weight="600">` + output.Markup + "</span>"
		output.Markup += msg.timestamp

		return output
	})

	return msg
}

func (m *FullMessage) Collapsed() bool { return false }

func (m *FullMessage) Unwrap(revert bool) *message.State {
	if revert {
		// Remove the handlers.
		m.unwrap()

		// Remove State's widgets from the containers.
		m.HeaderLabel.Destroy()
		m.MainBox.Remove(m.Content) // not ours, so don't destroy.

		// Remove the message from the grid.
		m.Avatar.Destroy()
		m.MainBox.Destroy()
	}

	return m.State
}

func formatLongTime(t time.Time) string {
	return `<span alpha="70%" size="small">` + humanize.TimeAgoLong(t) + `</span>`
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
