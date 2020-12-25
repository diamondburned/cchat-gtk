package httputil

type nopWriterImpl struct{}

var nopWriter = nopWriterImpl{}

func (nopWriterImpl) Write(b []byte) (int, error) { return len(b), nil }
func (nopWriterImpl) Close() error                { return nil }
