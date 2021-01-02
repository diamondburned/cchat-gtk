package auth

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/singlestack"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type RequestStack struct {
	*singlestack.Stack
	request *Request
	iconBox *gtk.Box
	icon    *gtk.Image
}

func NewRequestStack() *RequestStack {
	icon, _ := gtk.ImageNewFromIconName("document-edit-symbolic", gtk.ICON_SIZE_DIALOG)
	icon.SetHAlign(gtk.ALIGN_CENTER)
	icon.SetVAlign(gtk.ALIGN_CENTER)
	icon.SetHExpand(true)
	icon.SetVExpand(true)
	icon.SetOpacity(0.5)
	icon.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.SetHExpand(true)
	box.SetVExpand(true)
	box.Add(icon)
	box.Show()

	stack := singlestack.NewStack()
	stack.SetTransitionDuration(50)
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	stack.SetHExpand(true)
	stack.Add(box)

	return &RequestStack{
		Stack:   stack,
		iconBox: box,
		icon:    icon,
	}
}

// SetRequest sets the request into the stack. If auther is nil, then the
// placeholder icon is displayed. If auther is not nil, then Show() will be
// called.
func (rs *RequestStack) SetRequest(auther cchat.Authenticator, done func()) {
	if auther == nil {
		rs.request = nil
		rs.Stack.Add(rs.iconBox)
	} else {
		rs.request = NewRequest(auther, done)
		rs.request.Show()
		rs.Stack.Add(rs.request)
	}
}

func (rs *RequestStack) Request() *Request {
	return rs.request
}

// Request is a single page of authenticator fields.
type Request struct {
	*gtk.ScrolledWindow
	Box      *gtk.Box
	Grid     *gtk.Grid
	ErrRev   *gtk.Revealer
	ErrLabel *gtk.Label

	auther cchat.Authenticator
	labels []*gtk.Label
	texts  []Texter
}

func NewRequest(auther cchat.Authenticator, done func()) *Request {
	authEntries := auther.AuthenticateForm()

	errLabel, _ := gtk.LabelNew("")
	errLabel.SetUseMarkup(true)
	errLabel.SetMarginTop(8)
	errLabel.SetMarginStart(8)
	errLabel.SetMarginEnd(8)
	errLabel.SetLineWrap(true)
	errLabel.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	errLabel.Show()

	errRev, _ := gtk.RevealerNew()
	errRev.SetTransitionDuration(50)
	errRev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_DOWN)
	errRev.Add(errLabel)
	errRev.SetRevealChild(false)
	errRev.Show()

	grid, _ := gtk.GridNew()
	grid.SetRowSpacing(7)
	grid.SetColumnHomogeneous(false)
	grid.SetColumnSpacing(5)
	grid.SetMarginStart(12)
	grid.SetMarginEnd(12)
	grid.SetMarginTop(8)
	grid.Show()

	continueBtn, _ := gtk.ButtonNewWithLabel("Continue")
	continueBtn.SetHAlign(gtk.ALIGN_CENTER)
	continueBtn.Connect("clicked", func(*gtk.Button) { done() })
	continueBtn.SetBorderWidth(12)
	continueBtn.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(errRev, false, false, 0)
	box.PackStart(grid, true, true, 0)
	box.PackStart(continueBtn, false, false, 0)
	box.Show()

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetHExpand(true)
	sw.SetVExpand(true)
	sw.Add(box)

	req := &Request{
		ScrolledWindow: sw,
		Box:            box,
		Grid:           grid,
		ErrRev:         errRev,
		ErrLabel:       errLabel,

		auther: auther,
		labels: make([]*gtk.Label, len(authEntries)),
		texts:  make([]Texter, len(authEntries)),
	}

	for i, authEntry := range authEntries {
		label, texter := newEntry(authEntry)

		req.labels[i] = label
		req.texts[i] = texter

		grid.Attach(label, 0, i, 1, 1)
		grid.Attach(texter, 1, i, 1, 1)
	}

	return req
}

// SetError prints the error. If err is nil, then the label is cleared.
func (r *Request) SetError(err error) {
	var markup string
	if err != nil {
		builder := strings.Builder{}
		builder.WriteString(`<span color="red"><b>Error!</b>`)
		builder.WriteByte('\n')
		builder.WriteString(html.EscapeString(capitalizeFirst(err.Error())))
		builder.WriteString(`</span>`)
		markup = builder.String()
	}

	// Reveal if err is not nil.
	r.ErrRev.SetRevealChild(err != nil)
	r.ErrLabel.SetMarkup(markup)
}

// capitalizeFirst capitalizes the first letter.
func capitalizeFirst(str string) string {
	r, l := utf8.DecodeRuneInString(str)
	if l > 0 {
		return string(unicode.ToUpper(r)) + str[l:]
	}

	return str
}

func (r *Request) values() []string {
	var values = make([]string, len(r.texts))
	for i, texter := range r.texts {
		values[i] = texter.GetText()
	}

	return values
}

func newEntry(authEntry cchat.AuthenticateEntry) (*gtk.Label, Texter) {
	label, _ := gtk.LabelNew(authEntry.Name)
	label.SetXAlign(1) // right align
	label.SetJustify(gtk.JUSTIFY_RIGHT)
	label.SetLineWrap(true)
	label.Show()

	var texter Texter

	if authEntry.Multiline {
		texter = NewMultilineInput()
	} else {
		var input = NewEntryInput()
		if authEntry.Secret {
			input.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
			input.SetVisibility(false)
			input.SetInvisibleChar('●')
		} else {
			// usually; this is just an assumption
			input.SetInputPurpose(gtk.INPUT_PURPOSE_EMAIL)
		}

		texter = input
	}

	return label, texter
}

type Texter interface {
	gtk.IWidget
	GetText() string
	SetText(string)
}

type EntryInput struct {
	*gtk.Entry
}

var _ Texter = (*EntryInput)(nil)

func NewEntryInput() EntryInput {
	input, _ := gtk.EntryNew()
	input.SetVAlign(gtk.ALIGN_CENTER)
	input.SetHExpand(true)
	input.Show()

	return EntryInput{
		input,
	}
}

func (i EntryInput) GetText() (text string) {
	text, _ = i.Entry.GetText()
	return
}

type MultilineInput struct {
	*gtk.TextView
	Buffer *gtk.TextBuffer
}

var _ Texter = (*MultilineInput)(nil)

func NewMultilineInput() MultilineInput {
	view, _ := gtk.TextViewNew()
	view.SetWrapMode(gtk.WRAP_WORD_CHAR)
	view.SetEditable(true)
	view.SetHExpand(true)
	view.Show()

	buf, _ := view.GetBuffer()

	return MultilineInput{view, buf}
}

func (i MultilineInput) GetText() (text string) {
	start, end := i.Buffer.GetBounds()
	text, _ = i.Buffer.GetText(start, end, true)
	return
}

func (i MultilineInput) SetText(text string) {
	i.Buffer.SetText(text)
}
