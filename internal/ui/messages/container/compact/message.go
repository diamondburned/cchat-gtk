package compact

import (
	"time"

	"github.com/diamondburned/cchat-gtk/internal/humanize"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var messageTimeCSS = primitives.PrepareClassCSS("", `
	.message-time {
		margin: 0 8px;
	}
`)

var messageAuthorCSS = primitives.PrepareClassCSS("", `
	.message-author {
		margin-right: 0.5em;
	}
`)

type PresendMessage struct {
	message.Presender
	Message
}

func WrapPresendMessage(pstate *message.PresendState) PresendMessage {
	return PresendMessage{
		Presender: pstate,
		Message:   WrapMessage(pstate.State),
	}
}

type Message struct {
	*message.State
	Timestamp *gtk.Label
	Username  *labeluri.Label

	unwrap func()
}

var _ container.MessageRow = (*Message)(nil)

func WrapMessage(ct *message.State) Message {
	ts := message.NewTimestamp()
	ts.SetVAlign(gtk.ALIGN_START)
	ts.SetHAlign(gtk.ALIGN_END)
	ts.SetXAlign(1.00)
	ts.SetText(humanize.TimeAgo(ct.Time))
	ts.SetTooltipText(ct.Time.Format(time.Stamp))
	ts.Show()
	messageTimeCSS(ts)

	user := message.NewUsername()
	user.SetMaxWidthChars(22)
	user.SetHAlign(gtk.ALIGN_END)
	user.SetXAlign(1.0)
	user.SetJustify(gtk.JUSTIFY_RIGHT)
	user.SetEllipsize(pango.ELLIPSIZE_NONE)
	user.SetLineWrap(true)
	user.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	user.Show()
	messageAuthorCSS(user)

	ct.PackStart(ts, false, false, 0)
	ct.PackStart(user, false, false, 0)
	ct.PackStart(ct.Content, true, true, 0)
	ct.SetClass("compact")

	rcfg := markup.RenderConfig{}
	rcfg.NoReferencing = true
	rcfg.SetForegroundAnchor(ct.ContentBodyStyle)

	user.SetRenderer(func(rich text.Rich) markup.RenderOutput {
		return markup.RenderCmplxWithConfig(rich, rcfg)
	})

	return Message{
		State:     ct,
		Timestamp: ts,
		Username:  user,
		unwrap: ct.Author.Name.OnUpdate(func() {
			user.SetLabel(ct.Author.Name.Label())
		}),
	}
}

// SetReferenceHighlighter sets the reference highlighter into the message.
func (m Message) SetReferenceHighlighter(r labeluri.ReferenceHighlighter) {
	m.State.SetReferenceHighlighter(r)
	m.Username.SetReferenceHighlighter(r)
}

func (m Message) Revert() *message.State {
	m.unwrap()
	m.ClearBox()

	return m.Unwrap()
}
