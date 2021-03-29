package rich

import (
	"context"
	"html"
	"image"
	"runtime"
	"sync"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/pkg/errors"
)

// Small is a renderer that makes the plain text small.
func Small(content text.Rich) markup.RenderOutput {
	v := `<span size="small" alpha="50%">` + html.EscapeString(content.Content) + "</span>"
	return markup.RenderOutput{
		Markup: v,
		Input:  content.Content,
	}
}

// MakeRed is a renderer that makes the plain text red.
func MakeRed(content text.Rich) markup.RenderOutput {
	v := `<span color="red">` + html.EscapeString(content.Content) + `</span>`
	return markup.RenderOutput{
		Markup: v,
		Input:  content.Content,
	}
}

// LabelStateStorer is an interface for LabelState.
type LabelStateStorer interface {
	Label() text.Rich
	Image() LabelImage
	OnUpdate(func()) (remove func())
}

var _ LabelStateStorer = (*LabelState)(nil)

// NameContainer contains a reusable LabelState for cchat.Namer.
type NameContainer struct {
	LabelState
	state *containerState // for alignment
}

type containerState struct {
	// stop is the stored callback.
	stop func()
	// current is the context stopper being used.
	current context.CancelFunc
}

// Stop stops the name container. Calling Stop twice when no new namers are set
// will do nothing.
func (namec *NameContainer) Stop() {
	if namec.state != nil {
		namec.state.Stop()
		namec.LabelState.setLabel(text.Plain(""))
	}
}

func (state *containerState) Stop() {
	if state.current != nil {
		state.current()
		state.current = nil
	}

	if state.stop != nil {
		state.stop()
		state.stop = nil
	}
}

// QueueNamer tries using the namer in the background and queue the setter onto
// the next GLib loop iteration.
func (namec *NameContainer) QueueNamer(ctx context.Context, namer cchat.Namer) {
	if namec.state == nil {
		namec.state = &containerState{}
		runtime.SetFinalizer(namec.state, (*containerState).Stop)
	}

	namec.Stop()

	ctx, cancel := context.WithCancel(ctx)
	namec.state.current = cancel

	go func() {
		stop, err := namer.Name(ctx, namec)
		if err != nil {
			log.Error(errors.Wrap(err, "failed to activate namer"))
		}

		gts.ExecAsync(func() {
			namec.state.current()
			namec.state.current = nil
			namec.state.stop = stop
		})
	}()
}

// BindNamer binds a destructor signal to the name container to cancel a
// context.
func (namec *NameContainer) BindNamer(w primitives.Connector, sig string, namer cchat.Namer) {
	namec.QueueNamer(context.Background(), namer)

	// TODO: I have a hunch that everything below this will leak to hell. Just a
	// hunch.

	// namec.Stop()

	// ctx, cancel := context.WithCancel(context.Background())
	// namec.current = cancel

	// // TODO: this might leak, because namec.Stop references the fns list which
	// // might reference w indirectly.
	// w.Connect(sig, namec.Stop)

	// go func() {
	// 	stop, err := namer.Name(ctx, namec)
	// 	if err != nil {
	// 		log.Error(errors.Wrap(err, "failed to activate namer"))
	// 	}

	// 	gts.ExecAsync(func() {
	// 		namec.current()
	// 		namec.current = nil
	// 		namec.stop = stop // nil is OK.
	// 	})
	// }()
}

// LabelState provides a container for labels that allow other containers to
// extend upon. A zero-value instance is a valid instance.
type LabelState struct {
	// don't copy LabelState.
	_ [0]sync.Mutex

	label text.Rich

	fns    map[int]func()
	serial int
}

var _ cchat.LabelContainer = (*LabelState)(nil)

// NewLabelState creates a new label state.
func NewLabelState(l text.Rich) *LabelState {
	return &LabelState{label: l}
}

// String returns the inside label in plain text.
func (state *LabelState) String() string {
	return state.label.Content
}

// Label returns the inside label.
func (state *LabelState) Label() text.Rich {
	return state.label
}

// OnUpdate subscribes the given callback. The returned callback removes the
// given callback from the registry.
func (state *LabelState) OnUpdate(fn func()) (remove func()) {
	if state.fns == nil {
		state.fns = make(map[int]func(), 1)
	}

	id := state.serial
	state.fns[id] = fn
	state.serial++

	if !state.label.IsEmpty() {
		fn()
	}

	return func() { delete(state.fns, id) }
}

// SetLabel is called by cchat to update the state. The internal subscribed
// callbacks will be called in the main thread.
func (state *LabelState) SetLabel(label text.Rich) {
	gts.ExecAsync(func() { state.setLabel(label) })
}

func (state *LabelState) setLabel(label text.Rich) {
	state.label = label

	for _, fn := range state.fns {
		fn()
	}
}

// LabelImage is the first image from a label. If
type LabelImage struct {
	URL    string
	Text   string
	Size   image.Point
	Avatar bool
}

// HasImage returns true if the label has an image.
func (labelImage LabelImage) HasImage() bool {
	return labelImage.URL != ""
}

// Image returns the image, if any. Otherwise, an empty string is returned.
func (state *LabelState) Image() LabelImage {
	for _, segment := range state.label.Segments {
		if imager := segment.AsImager(); imager != nil {
			return LabelImage{
				URL:  imager.Image(),
				Text: imager.ImageText(),
				Size: image.Pt(imager.ImageSize()),
			}
		}

		if avatarer := segment.AsAvatarer(); avatarer != nil {
			size := avatarer.AvatarSize()

			return LabelImage{
				URL:    avatarer.Avatar(),
				Text:   avatarer.AvatarText(),
				Size:   image.Pt(size, size),
				Avatar: true,
			}
		}
	}

	return LabelImage{}
}
