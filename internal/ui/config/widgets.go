package config

import (
	"encoding/json"

	"github.com/gotk3/gotk3/gtk"
)

// EntryValue with JSON serde capabilities.
type EntryValue interface {
	json.Marshaler
	json.Unmarshaler
	Construct() gtk.IWidget
}

type _combo struct {
	selected *int
	options  []string
	change   func(int)
}

func Combo(selected *int, options []string, change func(int)) EntryValue {
	return &_combo{selected, options, change}
}

func (c *_combo) set(v int) {
	*c.selected = v
	if c.change != nil {
		c.change(v)
	}
}

func (c *_combo) Construct() gtk.IWidget {
	var combo, _ = gtk.ComboBoxTextNew()
	for _, opt := range c.options {
		combo.Append(opt, opt)
	}

	combo.Connect("changed", func(combo *gtk.ComboBoxText) { c.set(combo.GetActive()) })
	combo.SetActive(*c.selected)
	combo.SetHAlign(gtk.ALIGN_END)
	combo.Show()

	return combo
}

func (c *_combo) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c.selected)
}

func (c *_combo) UnmarshalJSON(b []byte) error {
	var value int
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	*c.selected = value
	return nil
}

type _switch struct {
	value  *bool
	change func(bool)
}

func Switch(value *bool, change func(bool)) EntryValue {
	return &_switch{value, change}
}

func (s *_switch) set(v bool) {
	*s.value = v
	if s.change != nil {
		s.change(v)
	}
}

func (s *_switch) Construct() gtk.IWidget {
	sw, _ := gtk.SwitchNew()
	sw.SetActive(*s.value)
	sw.Connect("notify::active", func(sw *gtk.Switch) { s.set(sw.GetActive()) })
	sw.SetHAlign(gtk.ALIGN_END)
	sw.Show()

	return sw
}

func (s *_switch) MarshalJSON() ([]byte, error) {
	return json.Marshal(*s.value)
}

func (s *_switch) UnmarshalJSON(b []byte) error {
	var value bool
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	s.set(value)
	return nil
}

type _inputentry struct {
	value  *string
	change func(string) error
}

func InputEntry(value *string, change func(string) error) EntryValue {
	return &_inputentry{value, change}
}

func (e *_inputentry) set(v string) error {
	*e.value = v
	if e.change != nil {
		return e.change(v)
	}
	return nil
}

func (e *_inputentry) Construct() gtk.IWidget {
	entry, _ := gtk.EntryNew()
	entry.SetHExpand(true)
	entry.SetText(*e.value)

	entry.Connect("changed", func(entry *gtk.Entry) {
		v, err := entry.GetText()
		if err != nil {
			return
		}

		if err := e.set(v); err != nil {
			entry.SetIconFromIconName(gtk.ENTRY_ICON_SECONDARY, "dialog-error")
			entry.SetIconTooltipText(gtk.ENTRY_ICON_SECONDARY, err.Error())
		} else {
			entry.RemoveIcon(gtk.ENTRY_ICON_SECONDARY)
		}
	})

	entry.Show()

	return entry
}

func (e *_inputentry) MarshalJSON() ([]byte, error) {
	return json.Marshal(*e.value)
}

func (e *_inputentry) UnmarshalJSON(b []byte) error {
	var value string
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	e.set(value)
	return nil
}
