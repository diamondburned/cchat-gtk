package auth

import (
	"html"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/dialog"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gtk"
)

type Dialog struct {
	*dialog.Modal
	Auther cchat.Authenticator
	onAuth func(cchat.Session)

	stack  *gtk.Stack // dialog stack
	scroll *gtk.ScrolledWindow
	body   *gtk.Box
	label  *gtk.Label

	// filled on spin()
	request *Request
}

// NewDialog makes a new authentication dialog. Auth() is called when the user
// is authenticated successfully inside the Gtk main thread.
func NewDialog(name text.Rich, auther cchat.Authenticator, auth func(cchat.Session)) *Dialog {
	label, _ := gtk.LabelNew("")
	label.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.Show()
	box.Add(label)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.Show()
	sw.Add(box)

	spinner, _ := gtk.SpinnerNew()
	spinner.Show()
	spinner.Start()
	spinner.SetSizeRequest(50, 50)

	stack, _ := gtk.StackNew()
	stack.Show()
	stack.SetVExpand(true)
	stack.SetHExpand(true)
	stack.AddNamed(sw, "main")
	stack.AddNamed(spinner, "spinner")

	d := &Dialog{
		Auther: auther,
		onAuth: auth,
		stack:  stack,
		scroll: sw,
		body:   box,
		label:  label,
	}
	d.Modal = dialog.NewModal(stack, "Log in to "+name.Content, "Log in", d.ok)
	d.Modal.SetDefaultSize(400, 300)
	d.spin(nil)
	d.Show()

	return d
}

func (d *Dialog) runOnAuth(ses cchat.Session) {
	// finalize
	d.Destroy()
	d.onAuth(ses)
}

func (d *Dialog) spin(err error) {
	// Remove old request.
	if d.request != nil {
		d.body.Remove(d.request)
	}

	// Print the error.
	if err != nil {
		d.label.SetMarkup(`<span color="red">` + html.EscapeString(err.Error()) + `</span>`)
	} else {
		d.label.SetText("")
	}

	// Restore the old widget states.
	d.stack.SetVisibleChildName("main")
	d.Dialog.SetSensitive(true)

	d.request = NewRequest(d.Auther.AuthenticateForm())
	d.body.Add(d.request)
}

func (d *Dialog) ok(m *dialog.Modal) {
	// Disable the buttons.
	d.Dialog.SetSensitive(false)

	// Switch to the spinner screen.
	d.stack.SetVisibleChildName("spinner")

	// Get the values of all fields.
	var values = d.request.values()

	gts.Async(func() (func(), error) {
		s, err := d.Auther.Authenticate(values)
		if err != nil {
			return func() { d.spin(err) }, nil
		}

		return func() { d.runOnAuth(s) }, nil
	})
}

type Request struct {
	*gtk.Grid
	labels  []*gtk.Label
	entries []Texter
}

func NewRequest(authEntries []cchat.AuthenticateEntry) *Request {
	grid, _ := gtk.GridNew()
	grid.Show()
	grid.SetRowSpacing(7)
	grid.SetColumnHomogeneous(true)
	grid.SetColumnSpacing(5)

	req := &Request{
		Grid:    grid,
		labels:  make([]*gtk.Label, len(authEntries)),
		entries: make([]Texter, len(authEntries)),
	}

	for i, authEntry := range authEntries {
		label, entry := newEntry(authEntry)

		req.labels[i] = label
		req.entries[i] = entry

		grid.Attach(label, 0, i, 1, 1)
		grid.Attach(entry, 1, i, 3, 1) // triple the width
	}

	return req
}

func (r *Request) values() []string {
	var values = make([]string, len(r.entries))
	for i, entry := range r.entries {
		values[i] = entry.GetText()
	}

	return values
}

func newEntry(authEntry cchat.AuthenticateEntry) (*gtk.Label, Texter) {
	label, _ := gtk.LabelNew(authEntry.Name)
	label.Show()
	label.SetXAlign(1) // right align
	label.SetJustify(gtk.JUSTIFY_RIGHT)
	label.SetLineWrap(true)

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
