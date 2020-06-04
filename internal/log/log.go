package log

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var globalBuffer struct {
	sync.Mutex
	entries  []Entry
	handlers []func(Entry)
}

func init() {
	AddEntryHandler(func(entry Entry) {
		fmt.Fprintln(os.Stderr, entry)
	})
}

type Entry struct {
	Time time.Time
	Msg  string
}

func (entry Entry) String() string {
	return entry.Time.Format(time.Stamp) + ": " + entry.Msg
}

// AddEntryHandler adds a handler, which will run asynchronously.
func AddEntryHandler(fn func(Entry)) {
	globalBuffer.handlers = append(globalBuffer.handlers, fn)
}

func Error(err error) {
	Write("Error: " + err.Error())
}

func Warn(err error) {
	Write("Warn: " + err.Error())
}

func Write(msg string) {
	WriteEntry(Entry{
		Time: time.Now(),
		Msg:  msg,
	})
}

func WriteEntry(entry Entry) {
	go func() {
		globalBuffer.Lock()
		globalBuffer.entries = append(globalBuffer.entries, entry)
		globalBuffer.Unlock()

		for _, fn := range globalBuffer.handlers {
			fn(entry)
		}
	}()
}
