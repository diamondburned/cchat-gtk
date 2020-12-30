package attachment

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

const (
	ThumbSize = 72
	IconSize  = 56
)

// File represents a middle format that can be used to create a
// MessageAttachment.
type File struct {
	Prog *Progress
	Name string
	Size int64 // -1 = stream
}

// NewFile creates a new attachment file with a progress state.
func NewFile(name string, size int64, open Open) File {
	return File{
		Prog: NewProgress(NewReusableReader(open), size),
		Name: name,
		Size: size,
	}
}

// AsAttachment turns File into a MessageAttachment. This method will always
// make a new MessageAttachment and will never return an old one.
//
// The reason being MessageAttachment should never be reused, as it hides the
// fact that the io.Reader is reusable.
func (f *File) AsAttachment() cchat.MessageAttachment {
	return cchat.MessageAttachment{
		Name:   f.Name,
		Reader: f.Prog,
	}
}

type Container struct {
	*gtk.Revealer
	Scroll *gtk.ScrolledWindow
	Box    *gtk.Box

	enabled bool

	// states
	files []File
	items map[string]primitives.WidgetDestroyer
}

var attachmentsCSS = primitives.PrepareCSS(`
	.attachments { padding: 5px; padding-bottom: 0 }
`)

func New() *Container {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	box.Show()

	primitives.AddClass(box, "attachments")
	primitives.AttachCSS(box, attachmentsCSS)

	scr, _ := gtk.ScrolledWindowNew(nil, nil)
	scr.SetPolicy(gtk.POLICY_EXTERNAL, gtk.POLICY_NEVER)
	scr.SetProperty("kinetic-scrolling", false)
	scr.Add(box)
	scr.Show()

	// Scroll left/right when the wheel goes up/down.
	scr.Connect("scroll-event", func(s *gtk.ScrolledWindow, ev *gdk.Event) bool {
		// Magic thing I found out while print-debugging. DeltaY shows the same
		// offset whether you scroll with Shift or not, which makes sense.
		// DeltaX is always 0.

		var adj = scr.GetHAdjustment()

		switch ev := gdk.EventScrollNewFromEvent(ev); ev.DeltaY() {
		case 1:
			adj.SetValue(adj.GetValue() + adj.GetStepIncrement())
		case -1:
			adj.SetValue(adj.GetValue() - adj.GetStepIncrement())
		default:
			// Not handled.
			return false
		}

		// Handled.
		return true
	})

	rev, _ := gtk.RevealerNew()
	rev.SetRevealChild(false)
	rev.Add(scr)

	return &Container{
		Revealer: rev,
		Scroll:   scr,
		Box:      box,
		items:    map[string]primitives.WidgetDestroyer{},
	}
}

// SetMarginStart sets the inner margin of the attachments carousel.
func (c *Container) SetMarginStart(margin int) {
	c.Box.SetMarginStart(margin)
}

// Enabled returns whether or not the container allows attachments.
func (c *Container) Enabled() bool {
	return c.enabled
}

func (c *Container) SetEnabled(enabled bool) {
	// Set the enabled state; reset the container if we're disabling the
	// attachment box.
	if c.enabled = enabled; !enabled {
		c.Reset()
	}
}

// Files returns the list of attachments
func (c *Container) Files() []File {
	return c.files
}

// Reset does NOT close files.
func (c *Container) Reset() {
	// Reset states. We do not touch the old files slice, as other callers may
	// be referencing and using it.
	c.files = nil

	// Clear all items.
	for _, item := range c.items {
		item.Destroy()
	}

	// Reset the map.
	c.items = map[string]primitives.WidgetDestroyer{}

	// Hide the window.
	c.SetRevealChild(false)
}

// AddFiles is used for the file chooser's callback.
func (c *Container) AddFiles(paths []string) {
	for _, path := range paths {
		if err := c.AddFile(path); err != nil {
			log.Error(errors.Wrap(err, "Failed to add file"))
		}
	}
}

// AddFile is used for the file picker.
func (c *Container) AddFile(path string) error {
	// Check the file and get the size.
	s, err := os.Stat(path)
	if err != nil {
		return errors.Wrap(err, "Failed to stat file")
	}

	var filename = c.append(
		filepath.Base(path), s.Size(),
		func() (io.ReadCloser, error) { return os.Open(path) },
	)

	scale := c.GetScaleFactor()

	// Maybe try making a preview. A nil image is fine, so we can skip the error
	// check.
	// TODO: add a filesize check
	pixbuf, _ := gdk.PixbufNewFromFileAtScale(path, ThumbSize*scale, ThumbSize*scale, true)
	c.addPreview(filename, thumbnailPixbuf(pixbuf, scale))
	return nil
}

// AddPixbuf is used for adding pixbufs from the clipboard.
func (c *Container) AddPixbuf(pb *gdk.Pixbuf) {
	var filename = c.append(
		fmt.Sprintf("clipboard_%d.png", len(c.files)+1), -1,
		func() (io.ReadCloser, error) {
			r, w := io.Pipe()
			go func() { w.CloseWithError(pb.WritePNG(w, 9)) }()
			return r, nil
		},
	)

	scale := c.GetScaleFactor()

	c.addPreview(filename, thumbnailPixbuf(pb, scale))
	return
}

// -- internal methods --

// append guarantees there's no collision. It returns the unique filename.
func (c *Container) append(name string, sz int64, open Open) string {
	// Show the preview window.
	c.SetRevealChild(true)

	// Guarantee that the filename will never collide.
	for _, file := range c.files {
		if file.Name == name {
			// Hopefully this works? I'm not sure. But this will keep prepending
			// an underscore.
			name = "_" + name
		}
	}

	c.files = append(c.files, NewFile(name, sz, open))
	return name
}

func (c *Container) remove(name string) {
	for i, file := range c.files {
		if file.Name == name {
			c.files = append(c.files[:i], c.files[i+1:]...)
			break
		}
	}

	if w, ok := c.items[name]; ok {
		w.Destroy()
		delete(c.items, name)
	}

	// Collapse the container if there's nothing.
	if len(c.items) == 0 {
		c.SetRevealChild(false)
	}
}

var previewCSS = primitives.PrepareCSS(`
	.attachment-preview {
		box-shadow: none;
		border: none;

		background-color: alpha(@theme_fg_color, 0.15);
		border-radius: 5px;
	}
`)

var deleteAttBtnCSS = primitives.PrepareCSS(`
	.delete-attachment {
		/* Remove styling from the Gtk themes */
		border: none;
		box-shadow: none;

		/* Add our own styling */
		border-radius: 999px 999px;
		transition: linear 100ms all;
		background-color: alpha(@theme_bg_color, 0.75);
	}
	.delete-attachment:hover {
		background-color: alpha(red, 0.5);
	}
`)

func (c *Container) addPreview(name string, thumbnail *cairo.Surface) {
	// Make a fallback image first.
	gimg, _ := roundimage.NewImage(4) // border-radius: 4px
	primitives.SetImageIcon(gimg.Image, iconFromName(name), IconSize)
	gimg.SetSizeRequest(ThumbSize, ThumbSize)
	gimg.SetVAlign(gtk.ALIGN_CENTER)
	gimg.SetHAlign(gtk.ALIGN_CENTER)
	gimg.SetTooltipText(name)
	gimg.Show()
	primitives.AddClass(gimg, "attachment-preview")
	primitives.AttachCSS(gimg, previewCSS)

	// Determine if we could generate an image preview.
	if thumbnail != nil {
		gimg.SetFromSurface(thumbnail)
	}

	// BLOAT!!! Make an overlay of an event box that, when hovered, will show
	// something that allows closing the image.
	del, _ := gtk.ButtonNewFromIconName("window-close-symbolic", gtk.ICON_SIZE_DIALOG)
	del.SetVAlign(gtk.ALIGN_CENTER)
	del.SetHAlign(gtk.ALIGN_CENTER)
	del.SetTooltipText("Remove " + name)
	del.Connect("clicked", func(del *gtk.Button) { c.remove(name) })
	del.Show()
	primitives.AddClass(del, "delete-attachment")
	primitives.AttachCSS(del, deleteAttBtnCSS)

	ovl, _ := gtk.OverlayNew()
	ovl.SetSizeRequest(ThumbSize, ThumbSize)
	ovl.Add(gimg)
	ovl.AddOverlay(del)
	ovl.Show()

	c.items[name] = ovl
	c.Box.PackStart(ovl, false, false, 0)
}

func thumbnailPixbuf(pixbuf *gdk.Pixbuf, scale int) *cairo.Surface {
	if pixbuf == nil {
		return nil
	}

	var (
		originalWidth  = pixbuf.GetWidth()
		originalHeight = pixbuf.GetHeight()

		scaledThumbSize           = ThumbSize * scale
		scaledWidth, scaledHeight = minsize(originalWidth, originalHeight, scaledThumbSize)

		// offset of src on thumbnail; one of those will be 0
		offsetX = float64(scaledThumbSize-scaledWidth) / 2
		offsetY = float64(scaledThumbSize-scaledHeight) / 2
	)

	thumbnail, err := gdk.PixbufNew(
		pixbuf.GetColorspace(),
		true, 8, // always have alpha, 8bpc
		scaledThumbSize, scaledThumbSize,
	)

	if err != nil {
		panic("failed to allocate upload thumbnail pixbuf: " + err.Error())
	}

	// Fill with transparent pixels.
	thumbnail.Fill(0x0)

	pixbuf.Scale(
		thumbnail,
		int(offsetX), int(offsetY),
		// size of src on thumbnail
		scaledWidth, scaledHeight,
		// no offset on source image
		offsetX, offsetY,
		// scale ratio for both sides
		float64(scaledWidth)/float64(originalWidth),
		float64(scaledHeight)/float64(originalHeight),
		// expensive rescale algorithm
		gdk.INTERP_HYPER,
	)

	surface, err := gdk.CairoSurfaceCreateFromPixbuf(thumbnail, scale, nil)
	if err != nil {
		panic("failed to create thumbnail cairo surface: " + err.Error())
	}

	return surface
}

func iconFromName(filename string) string {
	switch t := mime.TypeByExtension(filepath.Ext(filename)); {
	case strings.HasPrefix(t, "image"):
		return "image-x-generic-symbolic"

	case strings.HasPrefix(t, "audio"):
		return "audio-x-generic-symbolic"

	case strings.HasPrefix(t, "application"):
		return "application-x-appliance-symbolic"

	case strings.HasPrefix(t, "text"):
		fallthrough
	default:
		return "text-x-generic-symbolic"
	}
}

// minsize returns the scaled size so that the largest edge is maxsz.
func minsize(w, h, maxsz int) (int, int) {
	if w > h {
		// return the scaled width as max
		// h*max/w is the same as h/w*max but with more accuracy
		return maxsz, h * maxsz / w
	}

	return w * maxsz / h, maxsz
}

func min(w, h int) int {
	if w > h {
		return h
	}
	return w
}

func max(w, h int) int {
	if w > h {
		return w
	}
	return h
}
