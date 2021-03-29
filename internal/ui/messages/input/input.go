package input

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/container"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/attachment"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/username"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/message"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// Controller is an interface to control message containers.
type Controller interface {
	LatestMessageFrom(userID cchat.ID) container.MessageRow
	MessageAuthor(msgID cchat.ID) *message.Author
	Author(authorID cchat.ID) (name rich.LabelStateStorer)

	// SendMessage asynchronously sends the given message.
	SendMessage(msg message.PresendMessage)
}

// LabelBorrower is an interface that allows the caller to borrow a label.
type LabelBorrower interface {
	BorrowLabel(markup string)
	Unborrow()
}

type InputView struct {
	*Field
	Completer *completion.Completer
}

var textCSS = primitives.PrepareCSS(`
	.message-input {
		padding-top: 2px;
		padding-bottom: 2px;
	}

	.message-input, .message-input * {
		background-color: mix(@theme_bg_color, @theme_fg_color, 0.03);
	}

	.message-input * {
	    transition: linear 50ms background-color;
	    border: 1px solid alpha(@theme_fg_color, 0.2);
	    border-radius: 4px;
	}

	.message-input:focus * {
	    border-color: @theme_selected_bg_color;
	}
`)

var inputMainBoxCSS = primitives.PrepareClassCSS("input-box", `
	.input-box {
		background-color: @theme_bg_color;
	}
`)

func NewView(ctrl Controller, labeler LabelBorrower) *InputView {
	text, _ := gtk.TextViewNew()
	text.SetSensitive(false)
	text.SetWrapMode(gtk.WRAP_WORD_CHAR)
	text.SetVAlign(gtk.ALIGN_START)
	text.SetProperty("top-margin", 4)
	text.SetProperty("bottom-margin", 4)
	text.SetProperty("left-margin", 8)
	text.SetProperty("right-margin", 8)
	text.SetProperty("monospace", true)
	text.Show()

	primitives.AddClass(text, "message-input")
	primitives.AttachCSS(text, textCSS)

	// Bind the text event handler to text first.
	c := completion.NewCompleter(text)

	// Bind the input callback later.
	f := NewField(text, ctrl, labeler)
	f.Show()

	return &InputView{f, c}
}

func (v *InputView) SetMessenger(session cchat.Session, messenger cchat.Messenger) {
	v.Field.SetMessenger(session, messenger)

	if messenger == nil {
		v.Completer.SetCompleter(nil)
		return
	}

	// Ignore ok; completer can be nil.
	// TODO: this is possibly racy vs the above SetMessenger.
	var completer cchat.Completer
	if sender := messenger.AsSender(); sender != nil {
		completer = sender.AsCompleter()
	}

	v.Completer.SetCompleter(completer)
}

// wrapSpellCheck is a no-op but is replaced by gspell in ./spellcheck.go.
var wrapSpellCheck = func(textView *gtk.TextView) {}

const (
	sendButtonIcon  = "mail-send-symbolic"
	editButtonIcon  = "document-edit-symbolic"
	replyButtonIcon = "mail-reply-sender-symbolic"
	sendButtonSize  = gtk.ICON_SIZE_BUTTON

	ClampMaxSize   = 1000
	ClampThreshold = ClampMaxSize
)

type Field struct {
	*gtk.Box
	Clamp *handy.Clamp

	// MainBox contains the field box and the attachment container.
	MainBox     *gtk.Box
	Attachments *attachment.Container

	// FieldMainBox contains the username container and the input field. It spans
	// horizontally.
	FieldBox *gtk.Box
	Username *username.Container

	TextScroll *gtk.ScrolledWindow
	text       *gtk.TextView   // const
	buffer     *gtk.TextBuffer // const

	sendIcon *gtk.Image
	send     *gtk.Button

	attach *gtk.Button

	ctrl      Controller
	indicator LabelBorrower

	// Embed a state field which allows us to easily reset it.
	fieldState
}

type fieldState struct {
	UserID    string
	Messenger cchat.Messenger
	Sender    cchat.Sender
	upload    bool // true if server supports files
	editor    cchat.Editor
	typing    cchat.TypingIndicator

	replyingID cchat.ID
	editingID  cchat.ID

	lastTyped time.Time
	typerDura time.Duration
}

func (s *fieldState) Reset() {
	*s = fieldState{}
}

var inputFieldCSS = primitives.PrepareClassCSS("input-field", `
	.input-field {
		margin: 3px 5px;
		margin-top: 1px;
	}
`)

var scrolledInputCSS = primitives.PrepareClassCSS("scrolled-input", `
	.scrolled-input {
		margin: 0 5px;
	}
`)

func NewField(text *gtk.TextView, ctrl Controller, labeler LabelBorrower) *Field {
	field := &Field{
		text:      text,
		ctrl:      ctrl,
		indicator: labeler,
	}
	field.buffer, _ = text.GetBuffer()

	field.Username = username.NewContainer()
	field.Username.Show()

	field.TextScroll = scrollinput.NewV(text, 150)
	field.TextScroll.Show()
	scrolledInputCSS(field.TextScroll)

	field.attach, _ = gtk.ButtonNewFromIconName("mail-attachment-symbolic", sendButtonSize)
	field.attach.SetRelief(gtk.RELIEF_NONE)
	field.attach.SetSensitive(false)
	// Only show this if the server supports it (upload == true).
	primitives.AddClass(field.attach, "attach-button")

	field.sendIcon, _ = gtk.ImageNewFromIconName(sendButtonIcon, sendButtonSize)
	field.sendIcon.Show()

	field.send, _ = gtk.ButtonNew()
	field.send.SetImage(field.sendIcon)
	field.send.SetRelief(gtk.RELIEF_NONE)
	field.send.Show()
	primitives.AddClass(field.send, "send-button")

	field.FieldBox, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	field.FieldBox.PackStart(field.Username, false, false, 0)
	field.FieldBox.PackStart(field.attach, false, false, 0)
	field.FieldBox.PackStart(field.TextScroll, true, true, 0)
	field.FieldBox.PackStart(field.send, false, false, 0)
	field.FieldBox.Show()
	inputFieldCSS(field.FieldBox)

	field.Attachments = attachment.New()
	field.Attachments.Show()

	field.MainBox, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	field.MainBox.PackStart(field.Attachments, false, false, 0)
	field.MainBox.PackStart(field.FieldBox, false, false, 0)
	field.MainBox.Show()

	field.Clamp = handy.ClampNew()
	field.Clamp.SetMaximumSize(ClampMaxSize)
	field.Clamp.SetTighteningThreshold(ClampThreshold)
	field.Clamp.SetHExpand(true)
	field.Clamp.Add(field.MainBox)
	field.Clamp.Show()

	field.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	field.Box.Add(field.Clamp)
	field.Box.Show()
	inputMainBoxCSS(field.Clamp)

	text.SetFocusHAdjustment(field.TextScroll.GetHAdjustment())
	text.SetFocusVAdjustment(field.TextScroll.GetVAdjustment())
	// Bind text events.
	text.Connect("key-press-event", field.keyDown)
	// Bind the send button.
	field.send.Connect("clicked", func(*gtk.Button) { field.sendInput() })
	// Bind the attach button.
	field.attach.Connect("clicked", func(attach *gtk.Button) {
		gts.SpawnUploader("", field.Attachments.AddFiles)
	})

	// allocatedWidthGetter is used below.
	type allocatedWidthGetter interface {
		GetAllocatedWidth() int
	}

	// Connect to the field's revealer. On resize, we want the attachments
	// carousel to have the same padding too.
	field.Username.Connect("size-allocate", func(w allocatedWidthGetter) {
		// Calculate the left width: from the left of the message box to the
		// right of the attach button, covering the username container.
		var leftWidth = 5 + field.attach.GetAllocatedWidth() + w.GetAllocatedWidth()
		// Set the autocompleter's left margin to be the same.
		field.Attachments.SetMarginStart(leftWidth)
	})

	return field
}

// Reset prepares the field before SetSender() is called.
func (f *Field) Reset() {
	// Paranoia. The View should already change to a different stack, but we're
	// doing this just in case.
	f.text.SetSensitive(false)

	f.fieldState.Reset()
	f.Username.Reset()

	// reset the input
	f.clearText()
}

// SetMessenger changes the messenger of the input field. If nil, the input
// will be disabled. Reset() should be called first.
func (f *Field) SetMessenger(session cchat.Session, messenger cchat.Messenger) {
	// Update the left username container in the input.
	f.Username.Update(session, messenger)
	f.UserID = session.ID()

	// Set the sender.
	if messenger != nil {
		f.Messenger = messenger
		f.Sender = messenger.AsSender()
		f.text.SetSensitive(true)

		// Allow editor to be nil.
		f.editor = f.Messenger.AsEditor()
		// Allow typer to be nil.
		f.typing = f.Messenger.AsTypingIndicator()

		// See if we can upload files.
		f.SetAllowUpload(f.Sender != nil && f.Sender.CanAttach())

		// Populate the duration state if typer is not nil.
		if f.typing != nil {
			f.typerDura = f.typing.TypingTimeout()
		}
	}
}

func (f *Field) SetAllowUpload(allow bool) {
	f.upload = allow

	// Don't allow clicks on the attachment button if allow is false.
	f.attach.SetSensitive(allow)
	// Disable the attachmetn carousel for good measure, which also prevents
	// drag-and-drops.
	f.Attachments.SetEnabled(allow)

	// Show the attachment button if we can, else hide it.
	if f.upload {
		f.attach.Show()
	} else {
		f.attach.Hide()
	}
}

func (f *Field) StartReplyingTo(msgID cchat.ID) {
	// Clear the input to prevent mixing.
	f.clearText()

	f.replyingID = msgID
	f.sendIcon.SetFromIconName(replyButtonIcon, gtk.ICON_SIZE_BUTTON)

	if author := f.ctrl.MessageAuthor(msgID); author != nil {
		label := author.Name.Label()

		// Extract the name from the author's rich text and only render the area
		// with the MessageReference.
		for _, seg := range label.Segments {
			if seg.AsMessageReferencer() != nil || seg.AsMentioner() != nil {
				mention := markup.Render(markup.SubstringSegment(label, seg))
				f.indicator.BorrowLabel("Replying to " + mention)
				return
			}
		}
	}

	f.indicator.BorrowLabel("Replying to message.")
	return
}

// Editable returns whether or not the input field can be edited.
func (f *Field) Editable(msgID cchat.ID) bool {
	return f.editor != nil && f.editor.IsEditable(msgID)
}

func (f *Field) StartEditing(msgID cchat.ID) bool {
	// Do we support message editing? If not, exit.
	if !f.Editable(msgID) {
		return false
	}

	// Try and request the old message content for editing.
	content, err := f.editor.RawContent(msgID)
	if err != nil {
		// TODO: show error
		log.Error(errors.Wrap(err, "Failed to get message content"))
		return false
	}

	// Clear the input before editing to prevent mixing replying and editing
	// together.
	f.clearText()

	// Set the current editing state and set the input after requesting the
	// content.
	f.editingID = msgID
	f.buffer.SetText(content)

	f.indicator.BorrowLabel("Editing Message")
	f.sendIcon.SetFromIconName(editButtonIcon, sendButtonSize)

	return true
}

// StopEditing cancels the current editing message. It returns a false and does
// nothing if the editor is not editing anything.
func (f *Field) StopEditing() bool {
	if f.editingID == "" {
		return false
	}

	f.clearText()
	return true
}

// clearText resets the input field
func (f *Field) clearText() {
	f.editingID = ""
	f.replyingID = ""
	f.buffer.Delete(f.buffer.GetBounds())
	f.sendIcon.SetFromIconName(sendButtonIcon, sendButtonSize)
	f.indicator.Unborrow()
	f.Attachments.Reset()
}

// getText returns the text from the input, but it doesn't cut it.
func (f *Field) getText() string {
	start, end := f.buffer.GetBounds()
	text, _ := f.buffer.GetText(start, end, false)
	return text
}

func (f *Field) textLen() int {
	return f.buffer.GetCharCount()
}
