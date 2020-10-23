package singlestack

import (
	"github.com/gotk3/gotk3/gtk"
)

type Stack struct {
	*gtk.Stack
	current gtk.IWidget
}

func NewStack() *Stack {
	stack, _ := gtk.StackNew()
	return &Stack{stack, nil}
}

func (s *Stack) Add(w gtk.IWidget) {
	if s.current == w {
		return
	}

	if s.current != nil {
		s.Stack.Remove(s.current)
	}

	if w != nil {
		s.Stack.Add(w)
		s.Stack.SetVisibleChild(w)
	}

	s.current = w
}

func (s *Stack) SetVisibleChild(w gtk.IWidget) {
	s.Add(w)
}

func (s *Stack) GetChild() (gtk.IWidget, error) {
	return s.current, nil
}
