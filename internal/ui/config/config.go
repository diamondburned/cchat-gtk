// Package config provides the repository for configuration and preferences.
package config

import (
	"encoding/json"
	"sort"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/pkg/errors"
)

const ConfigFile = "config.json"

// List of config sections.
type Section uint8

const (
	Appearance Section = iota
	sectionLen
)

func (s Section) String() string {
	switch s {
	case Appearance:
		return "Appearance"
	default:
		return "???"
	}
}

type SectionEntries map[string]EntryValue

// UnmarshalJSON ignores all JSON entries with unknown keys.
func (s SectionEntries) UnmarshalJSON(b []byte) error {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}

	for k, entry := range s {
		v, ok := entries[k]
		if ok {
			if err := entry.UnmarshalJSON(v); err != nil {
				// Non-fatal error.
				log.Error(errors.Wrapf(err, "Failed to unmarshal key %q", k))
			}
		}
	}

	return nil
}

var sections = [sectionLen]SectionEntries{}

func AppearanceAdd(name string, value EntryValue) {
	sc := sections[Appearance]
	if sc == nil {
		sc = make(SectionEntries, 1)
		sections[Appearance] = sc
	}

	sc[name] = value
}

type Entry struct {
	Name  string
	Value EntryValue
}

func Sections() (sects [sectionLen][]Entry) {
	for i, section := range sections {
		var sect = make([]Entry, 0, len(section))
		for k, v := range section {
			sect = append(sect, Entry{k, v})
		}

		sort.Slice(sect, func(i, j int) bool {
			return sect[i].Name < sect[j].Name
		})

		sects[i] = sect
	}

	return
}

// Save the global config.
func Save() error {
	return MarshalToFile(ConfigFile, sections)
}

// Restore the global config. IsNotExist is not an error and will not be
// logged.
func Restore() {
	if err := UnmarshalFromFile(ConfigFile, &sections); err != nil {
		log.Error(errors.Wrap(err, "Failed to unmarshal main config.json"))
	}

	for path, v := range toRestore {
		if err := UnmarshalFromFile(path, v); err != nil {
			log.Error(errors.Wrapf(err, "Failed to unmarshal %s", path))
		}
	}
}

var toRestore = map[string]interface{}{}

// RegisterConfig adds the config filename into the registry of value pointers
// to unmarshal configs to.
func RegisterConfig(filename string, jsonValue interface{}) {
	toRestore[filename] = jsonValue
}

// Updaters contains a list of callbacks to be called when something is updated.
type Updaters []func()

func (us *Updaters) Add(f func()) {
	*us = append(*us, f)
}

func (us *Updaters) Updated() {
	for _, f := range *us {
		f()
	}
}
