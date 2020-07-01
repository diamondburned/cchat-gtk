package commander

import (
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/gotk3/gotk3/gtk"
)

type SessionCommander interface {
	cchat.Session
	cchat.Commander
}

type Buffer struct {
	*gtk.TextBuffer
	svcname string
	cmder   SessionCommander
}

// NewBuffer creates a new buffer with the given SessionCommander, or returns
// nil if cmder is nil.
func NewBuffer(svc cchat.Service, cmder SessionCommander) *Buffer {
	if cmder == nil {
		return nil
	}

	b, _ := gtk.TextBufferNew(nil)
	b.CreateTag("error", map[string]interface{}{
		"foreground": "#FF0000",
	})
	b.CreateTag("system", map[string]interface{}{
		"foreground": "#808080",
	})
	return &Buffer{b, svc.Name().Content, cmder}
}

// WriteError is not thread-safe.
func (b *Buffer) WriteError(err error) {
	b.InsertWithTagByName(b.GetEndIter(), err.Error()+"\n", "error")
}

// WriteSystem is not thread-safe.
func (b *Buffer) WriteSystem(bytes []byte) {
	b.InsertWithTagByName(b.GetEndIter(), string(bytes), "system")
}

// Printlnf is not thread-safe.
func (b *Buffer) Printlnf(f string, v ...interface{}) {
	b.WriteSystem([]byte(fmt.Sprintf(f+"\n", v...)))
}

// Write is thread-safe.
func (b *Buffer) Write(bytes []byte) (int, error) {
	gts.ExecAsync(func() { b.Insert(b.GetEndIter(), string(bytes)) })
	return len(bytes), nil
}

func (b *Buffer) ShowDialog() {
	SpawnDialog(b)
}
