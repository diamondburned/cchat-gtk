package actions

import (
	"log"
	"strings"

	"github.com/gotk3/gotk3/glib"
)

// Stateful is a stateful action group, which would allow additional methods
// that would otherwise be impossible to do with a simple Action Map.
type Stateful struct {
	glib.IActionGroup
	group  *glib.SimpleActionGroup
	labels []string // labels
}

func NewStateful() *Stateful {
	group := glib.SimpleActionGroupNew()
	return &Stateful{
		IActionGroup: group,
		group:        group,
	}
}

func (s *Stateful) Reset() {
	for _, label := range s.labels {
		s.group.RemoveAction(ActionName(label))
	}
	s.labels = nil
}

func (s *Stateful) AddAction(label string, call func()) {
	sa := glib.SimpleActionNew(ActionName(label), nil)
	sa.Connect("activate", call)

	s.labels = append(s.labels, label)
	s.group.AddAction(sa)
}

func (s *Stateful) LookupAction(label string) *glib.Action {
	for _, l := range s.labels {
		if l == label {
			return s.group.LookupAction(ActionName(label))
		}
	}
	return nil
}

func (s *Stateful) RemoveAction(label string) {
	for i, l := range s.labels {
		if l == label {
			s.labels = append(s.labels[:i], s.labels[:i+1]...)
			s.group.RemoveAction(ActionName(label))
			return
		}
	}
}

// ActionName converts the label name into the action name.
func ActionName(label string) (actionName string) {
	actionName = strings.Replace(label, " ", "-", -1)

	if !glib.ActionNameIsValid(actionName) {
		log.Panicf("Label makes for invalid action name %q\n", actionName)
	}

	return
}
