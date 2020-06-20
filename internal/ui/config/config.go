// Package config provides the repository for configuration and preferences.
package config

import "sort"

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

var Sections = [sectionLen][]Entry{}

func sortSection(section Section) {
	// TODO: remove the sorting and allow for declarative ordering
	sort.Slice(Sections[section], func(i, j int) bool {
		return Sections[section][i].Name < Sections[section][j].Name
	})
}

type Entry struct {
	Name  string
	Value EntryValue
}

func AppearanceAdd(name string, value EntryValue) {
	Sections[Appearance] = append(Sections[Appearance], Entry{
		Name:  name,
		Value: value,
	})
	sortSection(Appearance)
}
