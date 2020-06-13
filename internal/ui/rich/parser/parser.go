package parser

import (
	"bytes"
	"fmt"
	"html"
	"sort"

	"github.com/diamondburned/cchat/text"
)

type attrAppendMap struct {
	appended map[int]string
	indices  []int
}

func newAttrAppendedMap() attrAppendMap {
	return attrAppendMap{
		appended: make(map[int]string),
		indices:  []int{},
	}
}

func (a *attrAppendMap) add(ind int, attr string) {
	if _, ok := a.appended[ind]; ok {
		a.appended[ind] += attr
		return
	}

	a.appended[ind] = attr
	a.indices = append(a.indices, ind)
}

func (a attrAppendMap) get(ind int) string {
	return a.appended[ind]
}

func (a *attrAppendMap) finalize(strlen int) []int {
	// make sure there's always a closing tag at the end so the entire string
	// gets flushed.
	a.add(strlen, "")
	sort.Ints(a.indices)
	return a.indices
}

func RenderMarkup(content text.Rich) string {
	// Fast path.
	if len(content.Segments) == 0 {
		return html.EscapeString(content.Content)
	}

	buf := bytes.Buffer{}
	buf.Grow(len(content.Content))

	// // Sort so that all starting points are sorted incrementally.
	// sort.Slice(content.Segments, func(i, j int) bool {
	// 	i, _ = content.Segments[i].Bounds()
	// 	j, _ = content.Segments[j].Bounds()
	// 	return i < j
	// })

	// map to append strings to indices
	var appended = newAttrAppendedMap()

	// Parse all segments.
	for _, segment := range content.Segments {
		start, end := segment.Bounds()

		switch segment := segment.(type) {
		case text.Colorer:
			appended.add(start, fmt.Sprintf("<span color=\"#%06X\">", segment.Color()))
			appended.add(end, "</span>")
		}
	}

	var lastIndex = 0

	for _, index := range appended.finalize(len(content.Content)) {
		// Write the content.
		buf.WriteString(html.EscapeString(content.Content[lastIndex:index]))
		// Write the tags.
		buf.WriteString(appended.get(index))
		// Set the last index.
		lastIndex = index
	}

	return buf.String()
}
