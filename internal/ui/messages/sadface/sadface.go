// Package sadface provides different views for the message container.
package sadface

import (
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/gtk"
)

const FaceSize = 56

type WidgetUnreferencer interface {
	gtk.IWidget
	Unref()
}

type FaceView struct {
	gtk.Stack
	placeholder WidgetUnreferencer

	Face    *Container
	Loading *Spinner
}

func New(parent gtk.IWidget, placeholder WidgetUnreferencer) *FaceView {
	c := NewContainer()
	c.Show()

	s := NewSpinner()
	s.Show()

	// make an empty box for an empty page.
	b, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	stack, _ := gtk.StackNew()
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_CROSSFADE)
	stack.SetTransitionDuration(75)
	stack.AddNamed(parent, "main")
	stack.AddNamed(placeholder, "placeholder")
	stack.AddNamed(c, "face")
	stack.AddNamed(s, "loading")
	stack.AddNamed(b, "empty")

	// Show placeholder by default.
	stack.SetVisibleChildName("placeholder")

	return &FaceView{*stack, placeholder, c, s}
}

// Reset brings the view to an empty box.
func (v *FaceView) Reset() {
	v.ensurePlaceholderDestroyed()
	v.Loading.Spinner.Stop()
	v.Stack.SetVisibleChildName("empty")
}

// func (v *FaceView) Disable() {
// 	v.Stack.SetSensitive(false)
// }

// func (v *FaceView) Enable() {
// 	v.Stack.SetSensitive(true)
// }

func (v *FaceView) SetMain() {
	v.ensurePlaceholderDestroyed()
	v.Loading.Spinner.Stop()
	v.Stack.SetVisibleChildName("main")
}

func (v *FaceView) SetLoading() {
	v.ensurePlaceholderDestroyed()
	v.Loading.Spinner.Start()
	v.Stack.SetVisibleChildName("loading")
}

func (v *FaceView) SetError(err error) {
	v.Face.SetError(err)
	v.Stack.SetVisibleChildName("face")
	v.ensurePlaceholderDestroyed()
	v.Loading.Spinner.Stop()
}

func (v *FaceView) ensurePlaceholderDestroyed() {
	// If the placeholder is still there:
	if v.placeholder != nil {
		// Safely remove the placeholder from the stack.
		if v.Stack.GetVisibleChildName() == "placeholder" {
			v.Stack.SetVisibleChildName("empty")
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
	s.Start()
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
