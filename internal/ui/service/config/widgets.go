package config

import (
	"errors"
	"html"
	"sort"
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

const (
	vmargin = 8
	hmargin = 16
)

var monospaceCSS = primitives.PrepareCSS(`
	* {
		font-family: monospace;
	}
`)

type container struct {
	Grid      *gtk.Grid
	ErrHeader *errorHeader
	Entries   map[string]*entry

	fielderr *cchat.ErrInvalidConfigAtField
	apply    func() error
}

func newContainer(conf map[string]string, apply func() error) *container {
	var keys = make([]string, 0, len(conf))
	for k := range conf {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	errh := newErrorHeader()
	errh.Show()

	var grid, _ = gtk.GridNew()
	var entries = make(map[string]*entry, len(keys))

	var cc = &container{
		Grid:      grid,
		ErrHeader: errh,
		Entries:   entries,

		fielderr: &cchat.ErrInvalidConfigAtField{},
		apply:    apply,
	}

	for i, k := range keys {
		l, _ := gtk.LabelNew(k)
		l.SetHExpand(true)
		l.SetXAlign(0)
		l.Show()
		primitives.AttachCSS(l, monospaceCSS)

		e := newEntry(k, conf, cc.onEntryChange)
		e.Show()
		entries[k] = e

		grid.Attach(l, 0, i, 1, 1)
		grid.Attach(e, 1, i, 1, 1)
	}

	grid.SetRowHomogeneous(true)
	grid.SetRowSpacing(4)
	grid.SetColumnHomogeneous(true)
	grid.SetColumnSpacing(8)
	grid.Show()

	primitives.AddClass(grid, "config")

	return cc
}

func (c *container) onEntryChange() {
	err := c.apply()
	c.ErrHeader.SetError(err)

	// Reset the field error before unmarshaling into it again. If As() fails,
	// then all field errors will be cleared, as no keys will match.
	c.fielderr.Key = ""

	// fieldErred is true if the error is a field-specific error.
	var fieldErred = errors.As(err, &c.fielderr)

	// Loop over entries even if there is no field error. This clears up all
	// errors even if that is not the case.
	for k, entry := range c.Entries {
		if fieldErred && k == c.fielderr.Key {
			entry.SetError(c.fielderr.Err)
		} else {
			entry.SetError(nil)
		}
	}
}

type entry struct {
	*gtk.Entry
}

func newEntry(k string, conf map[string]string, change func()) *entry {
	e, _ := gtk.EntryNew()
	e.SetText(conf[k])
	e.SetHExpand(true)
	e.Connect("changed", func() {
		conf[k], _ = e.GetText()
		change()
	})
	primitives.AttachCSS(e, monospaceCSS)
	return &entry{e}
}

func (e *entry) SetError(err error) {
	if err != nil {
		e.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "dialog-error")
		e.SetIconTooltipText(gtk.ENTRY_ICON_SECONDARY, err.Error())
	} else {
		e.RemoveIcon(gtk.ENTRY_ICON_SECONDARY)
	}
}

type errorHeader struct {
	*gtk.Revealer
	l *gtk.Label
}

func newErrorHeader() *errorHeader {
	l, _ := gtk.LabelNew("")
	l.SetXAlign(0)
	l.Show()

	r, _ := gtk.RevealerNew()
	r.SetTransitionDuration(50)
	r.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_UP)
	r.SetRevealChild(false)
	r.SetMarginTop(vmargin)
	r.SetMarginBottom(vmargin)
	r.Add(l)

	return &errorHeader{r, l}
}

func (eh *errorHeader) SetError(err error) {
	if err != nil {
		// Cleanup the error message.
		parts := strings.Split(err.Error(), ": ")
		ermsg := parts[len(parts)-1]

		eh.SetRevealChild(true)
		eh.l.SetMarkup(`<span color="red">Error:</span> ` + html.EscapeString(ermsg))
		eh.l.SetTooltipText(err.Error())
	} else {
		eh.SetRevealChild(false)
		eh.l.SetText("")
		eh.l.SetTooltipText("")
	}
}
