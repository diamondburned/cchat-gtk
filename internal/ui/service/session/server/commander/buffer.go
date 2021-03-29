package commander

import (
	"bytes"
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/gotk3/gotk3/gtk"
)

// Buffer represents an unbuffered API around the text buffer.
type Buffer struct {
	*gtk.TextBuffer
	name  rich.LabelStateStorer
	cmder cchat.Commander
}

// NewBuffer creates a new buffer with the given SessionCommander, or returns
// nil if cmder is nil.
func NewBuffer(name rich.LabelStateStorer, cmder cchat.Commander) *Buffer {
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
	return &Buffer{b, name, cmder}
}

// WriteError is not thread-safe.
func (b *Buffer) WriteError(err error) {
	b.InsertWithTagByName(b.GetEndIter(), err.Error()+"\n", "error")
}

// WriteSystem is not thread-safe.
func (b *Buffer) WriteSystem(bytes []byte) {
	b.InsertWithTagByName(b.GetEndIter(), string(bytes), "system")
}

// Systemlnf is not thread-safe.
func (b *Buffer) Systemlnf(f string, v ...interface{}) {
	b.WriteSystem([]byte(fmt.Sprintf(f+"\n", v...)))
}

func (b *Buffer) WriteOutput(output []byte) {
	var iter = b.GetEndIter()

	b.Insert(iter, string(output))

	if !bytes.HasSuffix(output, []byte("\n")) {
		b.Insert(iter, "\n")
	}
}

func (b *Buffer) ShowDialog() {
	SpawnDialog(b)
}
