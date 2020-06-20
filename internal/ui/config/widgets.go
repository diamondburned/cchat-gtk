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

func (c *_combo) Construct() gtk.IWidget {
	var combo, _ = gtk.ComboBoxTextNew()
	for _, opt := range c.options {
		combo.Append(opt, opt)
	}

	combo.Connect("changed", func() {
		active := combo.GetActive()
		*c.selected = active

		if c.change != nil {
			c.change(active)
		}
	})

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

func (s *_switch) Construct() gtk.IWidget {
	sw, _ := gtk.SwitchNew()
	sw.SetActive(*s.value)

	sw.Connect("notify::active", func() {
		v := sw.GetActive()
		*s.value = v

		if s.change != nil {
			s.change(v)
		}
	})

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
	*s.value = value
	return nil
}
