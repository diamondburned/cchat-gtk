package attachment

import (
	"io"

	"github.com/pkg/errors"
)

type Open = func() (io.ReadCloser, error)

// ReusableReader provides an API which allows a reader to be used multiple
// times. It is NOT thread-safe to use.
type ReusableReader struct {
	open func() (io.ReadCloser, error)
	src  io.ReadCloser
}

var _ io.Reader = (*ReusableReader)(nil)

// NewReusableReader creates a new reader that is reusable after a read failure
// or a close. The given open() callback MUST be reproducible.
func NewReusableReader(open Open) *ReusableReader {
	return &ReusableReader{open, nil}
}

func (r *ReusableReader) Read(b []byte) (int, error) {
	if r.src == nil {
		o, err := r.open()
		if err != nil {
			return 0, errors.Wrap(err, "Failed to open reader")
		}
		r.src = o
	}

	n, err := r.src.Read(b)
	if err != nil { // err could be EOF or anything unexpected
		r.src.Close()
		r.src = nil
	}

	return n, err
}
