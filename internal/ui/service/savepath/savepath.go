package savepath

import (
	"bytes"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/service/session/server/traverse"
	"github.com/pkg/errors"
)

// map of services to a list of list of IDs.
var paths = make(pathMap)

type pathMap map[string]pathMap

const configName = "savepaths.json"

func init() {
	config.RegisterConfig(configName, &paths)
}

// ActiveSetter is an interface for all widgets that allow setting the active
// state.
type ActiveSetter interface {
	SetActive(bool)
}

// Restore restores the expand state by calling SetActive. This is meant to be
// used on a ToggledButton.
func Restore(b traverse.Breadcrumber, asetter ActiveSetter) {
	if IsExpanded(b) {
		asetter.SetActive(true)
	}
}

// IsExpanded returns true if the current breadcrumb node is expanded.
func IsExpanded(b traverse.Breadcrumber) bool {
	var path = traverse.TryID(b)
	var node = paths

	// Descend and traverse.
	var nest = 0
	for ; nest < len(path); nest++ {
		ch, ok := node[path[nest]]
		if !ok {
			return false
		}

		node = ch
	}

	// Return true if there is available a path that at least matches with the
	// breadcrumb path.
	return nest == len(path)
}

// SaveDelay is the delay to wait before saving.
const SaveDelay = 5 * time.Second

var lastSaved int64

// Save saves the list of paths. This function is not thread-safe. It is also
// non-blocking.
func Save() {
	var now = time.Now().UnixNano()

	if (lastSaved + int64(SaveDelay)) > now {
		return
	}

	lastSaved = now

	gts.AfterFunc(SaveDelay, func() {
		var buf bytes.Buffer

		// Marshal in the same thread to avoid race conditions.
		if err := config.PrettyMarshal(&buf, paths); err != nil {
			log.Error(errors.Wrap(err, "Failed to marshal paths"))
			return
		}

		go func() {
			if err := config.SaveToFile(configName, buf.Bytes()); err != nil {
				log.Error(errors.Wrap(err, "Failed to save paths"))
			}
		}()
	})
}

func Update(b traverse.Breadcrumber, expanded bool) {
	var path = traverse.TryID(b)
	var node = paths

	// TODO: this doesn't actually account for paths that no longer exist, but
	// it's complex to check.

	if expanded {
		// Descend and initialize.
		for i := 0; i < len(path); i++ {
			ch, ok := node[path[i]]
			if !ok {
				ch = make(pathMap)
				node[path[i]] = ch
			}

			node = ch
		}
	} else {
		for i := 0; i < len(path); i++ {
			ch, ok := node[path[i]]
			if !ok {
				// We can't find anything.
				return
			}

			if i == len(path)-1 {
				// We're at the last node, so we can delete things now.
				delete(node, path[i])
			}

			node = ch
		}
	}

	Save()
}
