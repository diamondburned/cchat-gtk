// Package sadface provides different views for the message container.
package sadface

import (
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

const FaceSize = 56

type FaceView struct {
	gtk.Stack
	placeholder gtk.IWidget

	face    *Container
	loading *Spinner
	parent  gtk.IWidget
	empty   gtk.IWidget
}

func New(parent gtk.IWidget, placeholder gtk.IWidget) *FaceView {
	c := NewContainer()
	c.Show()

	s := NewSpinner()
	s.Show()

	// make an empty box for an empty page.
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	stack, _ := gtk.StackNew()
	stack.SetTransitionDuration(55)
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	stack.Add(parent)
	stack.Add(c)
	stack.Add(s)
	stack.Add(b)

	// Show placeholder by default.
	stack.AddNamed(placeholder, "placeholder")
	stack.SetVisibleChild(placeholder)

	return &FaceView{
		Stack:       *stack,
		placeholder: placeholder,

		face:    c,
		loading: s,
		parent:  parent,
		empty:   b,
	}
}

// Reset brings the view to an empty box.
func (v *FaceView) Reset() {
	v.loading.Spinner.Stop()
	v.Stack.SetVisibleChild(v.empty)
	v.ensurePlaceholderDestroyed()
}

func (v *FaceView) SetMain() {
	v.loading.Spinner.Stop()
	v.Stack.SetVisibleChild(v.parent)
	v.ensurePlaceholderDestroyed()
}

func (v *FaceView) SetLoading() {
	v.loading.Spinner.Start()
	v.Stack.SetVisibleChild(v.loading)
	v.ensurePlaceholderDestroyed()
}

func (v *FaceView) SetError(err error) {
	v.face.SetError(err)
	v.Stack.SetVisibleChild(v.face)
	v.ensurePlaceholderDestroyed()
	v.loading.Spinner.Stop()
}

func (v *FaceView) ensurePlaceholderDestroyed() {
	// If the placeholder is still there:
	if v.placeholder != nil {
		// Safely remove the placeholder from the stack.
		if v.Stack.GetVisibleChildName() == "placeholder" {
			v.Stack.SetVisibleChild(v.empty)
		}

		// Remove the placeholder widget.
		v.Stack.Remove(v.placeholder)
		v.placeholder = nil
	}
}

type Spinner struct {
	gtk.Box
	Spinner *gtk.Spinner
}

func NewSpinner() *Spinner {
	s, _ := gtk.SpinnerNew()
	s.SetSizeRequest(FaceSize, FaceSize)
	s.Show()

	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	b.Add(s)
	b.SetHAlign(gtk.ALIGN_CENTER)
	b.SetVAlign(gtk.ALIGN_CENTER)

	return &Spinner{*b, s}
}

type Container struct {
	gtk.Box
	Face  *gtk.Image
	Error *gtk.Label
}

func NewContainer() *Container {
	face, _ := gtk.ImageNew()
	face.SetSizeRequest(FaceSize, FaceSize)
	face.Show()
	primitives.SetImageIcon(face, "face-sad-symbolic", FaceSize)

	errlabel, _ := gtk.LabelNew("")
	errlabel.SetOpacity(0.75) // low contrast good because unreadable
	errlabel.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	box.SetVAlign(gtk.ALIGN_CENTER)
	box.SetHAlign(gtk.ALIGN_CENTER)
	box.PackStart(face, false, false, 0)
	box.PackStart(errlabel, false, false, 0)

	return &Container{*box, face, errlabel}
}

// SetError sets the view to display the error. Error must not be nil.
func (v *Container) SetError(err error) {
	// Split the error.
	parts := strings.Split(err.Error(), ": ")
	v.Error.SetLabel("Error: " + parts[len(parts)-1])

	// Use the full error for the tooltip.
	v.Box.SetTooltipText(err.Error())
}
