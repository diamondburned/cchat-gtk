package attachment

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/disintegration/imaging"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

var pngEncoder = png.Encoder{
	CompressionLevel: png.BestCompression,
}

const FileIconSize = 72

// File represents a middle format that can be used to create a
// MessageAttachment.
type File struct {
	Prog *Progress
	Name string
	Size int64
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

	// states
	files []File
	items map[string]gtk.IWidget
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
		items:    map[string]gtk.IWidget{},
	}
}

// SetMarginStart sets the inner margin of the attachments carousel.
func (c *Container) SetMarginStart(margin int) {
	c.Box.SetMarginStart(margin)
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
		c.Box.Remove(item)
	}

	// Reset the map.
	c.items = map[string]gtk.IWidget{}

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

	// Maybe try making a preview. A nil image is fine, so we can skip the error
	// check.
	// TODO: add a filesize check
	i, _ := imaging.Open(path, imaging.AutoOrientation(true))
	c.addPreview(filename, i)

	return nil
}

// AddPixbuf is used for adding pixbufs from the clipboard.
func (c *Container) AddPixbuf(pb *gdk.Pixbuf) error {
	// Pixbuf's colorspace is only RGB. This is indicated with
	// GDK_COLORSPACE_RGB.
	if pb.GetColorspace() != gdk.COLORSPACE_RGB {
		return errors.New("Pixbuf has unsupported colorspace")
	}

	// Assert that the pixbuf has alpha, as we're using RGBA.
	if !pb.GetHasAlpha() {
		return errors.New("Pixbuf has no alpha channel")
	}

	// Assert that there are 4 channels: red, green, blue and alpha.
	if pb.GetNChannels() != 4 {
		return errors.New("Pixbuf has unexpected channel count")
	}

	// Assert that there are 8 bits in a channel/sample.
	if pb.GetBitsPerSample() != 8 {
		return errors.New("Pixbuf has unexpected bits per sample")
	}

	var img = &image.NRGBA{
		Pix:    pb.GetPixels(),
		Stride: pb.GetRowstride(),
		Rect:   image.Rect(0, 0, pb.GetWidth(), pb.GetHeight()),
	}

	// Store the image in memory.
	var buf bytes.Buffer

	if err := pngEncoder.Encode(&buf, img); err != nil {
		return errors.Wrap(err, "Failed to encode PNG")
	}

	var filename = c.append(
		fmt.Sprintf("clipboard_%d.png", len(c.files)+1), int64(buf.Len()),
		func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
		},
	)

	c.addPreview(filename, img)

	return nil
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
		c.Box.Remove(w)
		delete(c.items, name)
	}

	// Collapse the container if there's nothing.
	if len(c.items) == 0 {
		c.SetRevealChild(false)
	}
}

var previewCSS = primitives.PrepareCSS(`
	.attachment-preview {
		background-color: alpha(@theme_fg_color, 0.2);
		border-radius: 4px;
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
		background-color: alpha(@theme_bg_color, 0.50);
	}
	.delete-attachment:hover {
		background-color: alpha(red, 0.5);
	}
`)

func (c *Container) addPreview(name string, src image.Image) {
	// Make a fallback image first.
	gimg, _ := gtk.ImageNew()
	primitives.SetImageIcon(gimg, "image-x-generic-symbolic", FileIconSize/3)
	gimg.SetSizeRequest(FileIconSize, FileIconSize)
	gimg.SetVAlign(gtk.ALIGN_CENTER)
	gimg.SetHAlign(gtk.ALIGN_CENTER)
	gimg.SetTooltipText(name)
	gimg.Show()
	primitives.AddClass(gimg, "attachment-preview")
	primitives.AttachCSS(gimg, previewCSS)

	// Determine if we could generate an image preview.
	if src != nil {
		// Get the minimum dimension.
		var w, h = minsize(src.Bounds().Dx(), src.Bounds().Dy(), FileIconSize)

		var img *image.NRGBA
		// Downscale the image.
		img = imaging.Resize(src, w, h, imaging.Lanczos)

		// Crop to a square.
		img = imaging.CropCenter(img, FileIconSize, FileIconSize)

		// Copy the image to a pixbuf.
		gimg.SetFromPixbuf(gts.RenderPixbuf(img))
	}

	// BLOAT!!! Make an overlay of an event box that, when hovered, will show
	// something that allows closing the image.
	del, _ := gtk.ButtonNewFromIconName("window-close", gtk.ICON_SIZE_DIALOG)
	del.SetVAlign(gtk.ALIGN_CENTER)
	del.SetHAlign(gtk.ALIGN_CENTER)
	del.SetTooltipText("Remove " + name)
	del.Connect("clicked", func() { c.remove(name) })
	del.Show()
	primitives.AddClass(del, "delete-attachment")
	primitives.AttachCSS(del, deleteAttBtnCSS)

	ovl, _ := gtk.OverlayNew()
	ovl.SetSizeRequest(FileIconSize, FileIconSize)
	ovl.Add(gimg)
	ovl.AddOverlay(del)
	ovl.Show()

	c.items[name] = ovl
	c.Box.PackStart(ovl, false, false, 0)
}

func minsize(w, h, maxsz int) (int, int) {
	if w < h {
		// return the scaled width as max
		// h*max/w is the same as h/w*max but with more accuracy
		return maxsz, h * maxsz / w
	}

	return w * maxsz / h, maxsz
}
