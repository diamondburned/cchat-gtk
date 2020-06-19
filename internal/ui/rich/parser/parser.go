package parser

import (
	"fmt"
	"sort"
	"time"

	"github.com/diamondburned/cchat-gtk/internal/log"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

func AppendEditBadge(b *gtk.TextBuffer, editedAt time.Time) {
	r := newRenderCtx(b)

	t := r.createTag(map[string]interface{}{
		"scale":      0.84,
		"scale-set":  true,
		"foreground": "#808080", // blue-ish URL color
	})

	bindClicker(t, func(_ *gtk.TextView, ev *gdk.Event) {
		switch ev := gdk.EventMotionNewFromEvent(ev); ev.Type() {
		case gdk.EVENT_PROXIMITY_IN:
			log.Println("Proximity in")
		case gdk.EVENT_PROXIMITY_OUT:
			log.Println("Proximity out")
		}
	})

	b.InsertWithTag(b.GetEndIter(), " (edited)", t)
}

func RenderTextBuffer(b *gtk.TextBuffer, content text.Rich) {
	r := newRenderCtx(b)
	b.SetText(content.Content)

	// Sort so that all starting points are sorted incrementally.
	sort.Slice(content.Segments, func(i, j int) bool {
		i, _ = content.Segments[i].Bounds()
		j, _ = content.Segments[j].Bounds()
		return i < j
	})

	for _, segment := range content.Segments {
		start, end := segment.Bounds()

		switch segment := segment.(type) {
		case text.Attributor:
			r.tagAttr(start, end, segment.Attribute())

		case text.Colorer:
			color := fmt.Sprintf("#%06X", segment.Color())
			r.applyProps(start, end, map[string]interface{}{
				"foreground": color,
			})

		case text.Codeblocker:
			r.applyProps(start, end, map[string]interface{}{
				"family": "Monospace",
			})
		}
	}
}

type renderCtx struct {
	b *gtk.TextBuffer
	t *gtk.TextTagTable
}

func newRenderCtx(b *gtk.TextBuffer) *renderCtx {
	t, _ := b.GetTagTable()
	return &renderCtx{b, t}
}

type OnClicker func(tv *gtk.TextView, ev *gdk.Event)

func bindClicker(v primitives.Connector, fn OnClicker) {
	v.Connect("event", func(_ *gtk.TextTag, tv *gtk.TextView, ev *gdk.Event) {
		evButton := gdk.EventButtonNewFromEvent(ev)
		if evButton.Type() != gdk.EVENT_BUTTON_RELEASE || evButton.Button() != gdk.BUTTON_PRIMARY {
			return
		}

		fn(tv, ev)
	})
}

func (r *renderCtx) applyHyperlink(start, end int, url string) {
	t := r.createTag(map[string]interface{}{
		"underline":  pango.UNDERLINE_SINGLE,
		"foreground": "#3F7CE0", // blue-ish URL color
	})

	bindClicker(t, func(*gtk.TextView, *gdk.Event) {
		if err := open.Start(url); err != nil {
			log.Error(errors.Wrap(err, "Failed to open image URL"))
		}
	})
}

func (r *renderCtx) applyProps(start, end int, props map[string]interface{}) {
	tag := r.createTag(props)
	r.applyTag(start, end, tag)
}

func (r *renderCtx) applyTag(start, end int, tag *gtk.TextTag) {
	istart, iend := r.iters(start, end)
	r.b.ApplyTag(tag, istart, iend)
}

func (r *renderCtx) createTag(props map[string]interface{}) *gtk.TextTag {
	t, _ := gtk.TextTagNew("")
	r.t.Add(t)

	if props != nil {
		for k, v := range props {
			t.SetProperty(k, v)
		}
	}

	return t
}

func (r *renderCtx) iters(start, end int) (is, ie *gtk.TextIter) {
	return r.b.GetIterAtOffset(start), r.b.GetIterAtOffset(end)
}

func (r *renderCtx) tagAttr(start, end int, attr text.Attribute) {
	var props = tagAttrMap(attr)
	if props == nil {
		return
	}

	r.applyTag(start, end, r.createTag(props))
}

func tagAttrMap(attr text.Attribute) map[string]interface{} {
	if attr == 0 {
		return nil
	}

	var props = make(map[string]interface{}, 1)

	if attr.Has(text.AttrBold) {
		props["weight"] = pango.WEIGHT_BOLD
	}
	if attr.Has(text.AttrItalics) {
		props["style"] = pango.STYLE_ITALIC
	}
	if attr.Has(text.AttrUnderline) {
		props["underline"] = pango.UNDERLINE_SINGLE
	}
	if attr.Has(text.AttrStrikethrough) {
		props["strikethrough"] = true
	}
	if attr.Has(text.AttrSpoiler) {
		props["foreground"] = "#808080"
	}
	if attr.Has(text.AttrMonospace) {
		props["family"] = "Monospace"
	}

	return props
}
