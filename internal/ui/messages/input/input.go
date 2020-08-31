package input

import (
	"time"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/attachment"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/completion"
	"github.com/diamondburned/cchat-gtk/internal/ui/messages/input/username"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/scrollinput"
	"github.com/diamondburned/gspell"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

// Controller is an interface to control message containers.
type Controller interface {
	AddPresendMessage(msg PresendMessage) (onErr func(error))
	LatestMessageFrom(userID string) (messageID string, ok bool)
}

type InputView struct {
	*Field
	Completer *completion.View
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

var inputBoxCSS = primitives.PrepareClassCSS("input-box", `
	.input-box {
		background-color: @theme_bg_color;
	}
`)

func NewView(ctrl Controller) *InputView {
	text, _ := gtk.TextViewNew()
	text.SetSensitive(false)
	text.SetWrapMode(gtk.WRAP_WORD_CHAR)
	text.SetVAlign(gtk.ALIGN_START)
	text.SetProperty("top-margin", 4)
	text.SetProperty("bottom-margin", 4)
	text.SetProperty("left-margin", 8)
	text.SetProperty("right-margin", 8)
	text.Show()

	primitives.AddClass(text, "message-input")
	primitives.AttachCSS(text, textCSS)

	// Bind the text event handler to text first.
	c := completion.New(text)

	// Bind the input callback later.
	f := NewField(text, ctrl)
	f.Show()

	return &InputView{f, c}
}

func (v *InputView) SetSender(session cchat.Session, sender cchat.ServerMessageSender) {
	v.Field.SetSender(session, sender)

	// Ignore ok; completer can be nil.
	completer, _ := sender.(cchat.ServerMessageSendCompleter)
	v.Completer.SetCompleter(completer)
}

type Field struct {
	// Box contains the field box and the attachment container.
	*gtk.Box
	Attachments *attachment.Container

	// FieldBox contains the username container and the input field. It spans
	// horizontally.
	FieldBox *gtk.Box
	Username *username.Container

	TextScroll *gtk.ScrolledWindow
	text       *gtk.TextView    // const
	speller    *gspell.TextView // const
	buffer     *gtk.TextBuffer  // const

	send   *gtk.Button
	attach *gtk.Button

	ctrl Controller

	// Embed a state field which allows us to easily reset it.
	fieldState
}

type fieldState struct {
	UserID string
	Sender cchat.ServerMessageSender
	upload bool // true if server supports files
	editor cchat.ServerMessageEditor
	typer  cchat.ServerMessageTypingIndicator

	editingID string // never empty
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

func NewField(text *gtk.TextView, ctrl Controller) *Field {
	field := &Field{text: text, ctrl: ctrl}
	field.buffer, _ = text.GetBuffer()
	field.speller = gspell.GetFromGtkTextView(text)
	field.speller.BasicSetup()

	field.Username = username.NewContainer()
	field.Username.Show()

	field.TextScroll = scrollinput.NewV(text, 150)
	field.TextScroll.Show()
	scrolledInputCSS(field.TextScroll)

	field.attach, _ = gtk.ButtonNewFromIconName("mail-attachment-symbolic", gtk.ICON_SIZE_BUTTON)
	field.attach.SetRelief(gtk.RELIEF_NONE)
	field.attach.SetSensitive(false)
	// Only show this if the server supports it (upload == true).
	primitives.AddClass(field.attach, "attach-button")

	field.send, _ = gtk.ButtonNewFromIconName("mail-send-symbolic", gtk.ICON_SIZE_BUTTON)
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

	field.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	field.Box.PackStart(field.Attachments, false, false, 0)
	field.Box.PackStart(field.FieldBox, false, false, 0)
	field.Box.Show()
	inputBoxCSS(field.Box)

	text.SetFocusHAdjustment(field.TextScroll.GetHAdjustment())
	text.SetFocusVAdjustment(field.TextScroll.GetVAdjustment())
	// Bind text events.
	text.Connect("key-press-event", field.keyDown)
	// Bind the send button.
	field.send.Connect("clicked", field.sendInput)
	// Bind the attach button.
	field.attach.Connect("clicked", func() { gts.SpawnUploader("", field.Attachments.AddFiles) })

	// Connect to the field's revealer. On resize, we want the attachments
	// carousel to have the same padding too.
	field.Username.Connect("size-allocate", func(w gtk.IWidget) {
		// Calculate the left width: from the left of the message box to the
		// right of the attach button, covering the username container.
		var leftWidth = 5 + field.attach.GetAllocatedWidth() + w.ToWidget().GetAllocatedWidth()
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

// SetSender changes the sender of the input field. If nil, the input will be
// disabled. Reset() should be called first.
func (f *Field) SetSender(session cchat.Session, sender cchat.ServerMessageSender) {
	// Update the left username container in the input.
	f.Username.Update(session, sender)
	f.UserID = session.ID()

	// Set the sender.
	if sender != nil {
		f.Sender = sender
		f.text.SetSensitive(true)

		// Allow editor to be nil.
		f.editor, _ = sender.(cchat.ServerMessageEditor)
		// Allow typer to be nil.
		f.typer, _ = sender.(cchat.ServerMessageTypingIndicator)

		// See if we can upload files.
		_, allowUpload := sender.(cchat.ServerMessageAttachmentSender)
		f.SetAllowUpload(allowUpload)

		// Populate the duration state if typer is not nil.
		if f.typer != nil {
			f.typerDura = f.typer.TypingTimeout()
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

// Editable returns whether or not the input field can be edited.
func (f *Field) Editable(msgID string) bool {
	return f.editor != nil && f.editor.MessageEditable(msgID)
}

func (f *Field) StartEditing(msgID string) bool {
	// Do we support message editing? If not, exit.
	if !f.Editable(msgID) {
		return false
	}

	// Try and request the old message content for editing.
	content, err := f.editor.RawMessageContent(msgID)
	if err != nil {
		// TODO: show error
		log.Error(errors.Wrap(err, "Failed to get message content"))
		return false
	}

	// Set the current editing state and set the input after requesting the
	// content.
	f.editingID = msgID
	f.buffer.SetText(content)

	return true
}

// StopEditing cancels the current editing message. It returns a false and does
// nothing if the editor is not editing anything.
func (f *Field) StopEditing() bool {
	if f.editingID == "" {
		return false
	}

	f.editingID = ""
	f.clearText()

	return true
}

// clearText resets the input field
func (f *Field) clearText() {
	f.buffer.Delete(f.buffer.GetBounds())
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
