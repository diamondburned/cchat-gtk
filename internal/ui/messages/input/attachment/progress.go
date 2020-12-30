package attachment

import (
	"errors"
	"io"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

type MessageUploader struct {
	*gtk.Grid
}

// NewMessageUploader creates a new MessageUploader. It returns nil if there are
// no files.
func NewMessageUploader(files []File) *MessageUploader {
	m := &MessageUploader{}

	m.Grid, _ = gtk.GridNew()
	m.Grid.SetHExpand(true)
	m.Grid.SetColumnSpacing(4)
	m.Grid.SetRowSpacing(2)
	m.Grid.SetRowHomogeneous(true)

	primitives.AddClass(m.Grid, "upload-progress")

	for i, file := range files {
		var pbar = NewProgressBar(file)

		m.Grid.Attach(pbar.Name, 0, i, 1, 1)
		m.Grid.Attach(pbar.PBar, 1, i, 1, 1)
	}

	return m
}

type ProgressBar struct {
	PBar *gtk.ProgressBar
	Name *gtk.Label
}

func NewProgressBar(file File) *ProgressBar {
	bar, _ := gtk.ProgressBarNew()
	bar.SetVAlign(gtk.ALIGN_CENTER)
	bar.Show()

	var label = file.Name
	if file.Size > 0 {
		label += " - " + glib.FormatSize(uint64(file.Size))
	}

	name, _ := gtk.LabelNew(label)
	name.SetMaxWidthChars(45)
	name.SetSingleLineMode(true)
	name.SetEllipsize(pango.ELLIPSIZE_MIDDLE)
	name.SetXAlign(1)
	name.Show()

	// Override the upload read callback.
	file.Prog.u = func(fraction float64) {
		gts.ExecAsync(func() {
			if fraction == -1 {
				// Pulse the bar around, as we don't know the total bytes.
				bar.Pulse()
			} else {
				// We know the progress, so use the percentage.
				bar.SetFraction(fraction)
			}
		})
	}

	return &ProgressBar{bar, name}
}

// Progress wraps around a ReadCloser and implements a progress state for a
// reader.
type Progress struct {
	u func(float64) // read callback, arg is percentage
	r io.Reader
	s float64 // total, const
	n uint64  // cumulative
}

// NewProgress creates a new upload progress state.
func NewProgress(r io.Reader, size int64) *Progress {
	return &Progress{
		r: r,
		s: float64(size),
		n: 0,
	}
}

// frac returns the current percentage, or -1 is there is no total.
func (p *Progress) frac() float64 {
	if p.s > 0 {
		return float64(p.n) / p.s
	}
	return -1
}

func (p *Progress) Read(b []byte) (int, error) {
	// Read and cumulate total bytes read if there are no errors or if the error
	// is not fatal (EOF).
	n, err := p.r.Read(b)
	if err == nil || errors.Is(err, io.EOF) {
		p.n += uint64(n)
	} else {
		// If we have an unexpected error, then we should reset the bytes read
		// to 0.
		p.n = 0
	}

	if p.u != nil {
		p.u(p.frac())
	}

	return n, err
}
