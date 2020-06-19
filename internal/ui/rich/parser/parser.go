package parser

import (
	"bytes"
	"fmt"
	"html"
	"sort"
	"strings"

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

func (a *attrAppendMap) span(start, end int, attr string) {
	a.add(start, `<span `+attr+`>`)
	a.add(end, "</span>")
}

func (a *attrAppendMap) pair(start, end int, open, close string) {
	a.add(start, open)
	a.add(end, close)
}

func (a *attrAppendMap) addf(ind int, f string, argv ...interface{}) {
	a.add(ind, fmt.Sprintf(f, argv...))
}

func (a *attrAppendMap) pad(ind int) {
	a.add(ind, "\n")
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

func markupAttr(attr text.Attribute) string {
	// meme fast path
	if attr == 0 {
		return ""
	}

	var attrs = make([]string, 0, 1)
	if attr.Has(text.AttrBold) {
		attrs = append(attrs, `weight="bold"`)
	}
	if attr.Has(text.AttrItalics) {
		attrs = append(attrs, `style="italic"`)
	}
	if attr.Has(text.AttrUnderline) {
		attrs = append(attrs, `underline="single"`)
	}
	if attr.Has(text.AttrStrikethrough) {
		attrs = append(attrs, `strikethrough="true"`)
	}
	if attr.Has(text.AttrSpoiler) {
		attrs = append(attrs, `foreground="#808080"`) // no fancy click here
	}
	if attr.Has(text.AttrMonospace) {
		attrs = append(attrs, `font_family="monospace"`)
	}
	return strings.Join(attrs, " ")
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
		case text.Linker:
			appended.addf(start, `<a href="%s">`, html.EscapeString(segment.Link()))
			appended.add(end, "</a>")

		case text.Imager:
			// Ends don't matter with images.
			appended.addf(start,
				`<a href="%s">%s</a>`,
				html.EscapeString(segment.Image()), html.EscapeString(segment.ImageText()),
			)

		case text.Colorer:
			appended.span(start, end, fmt.Sprintf(`color="#%06X"`, segment.Color()))

		case text.Attributor:
			appended.span(start, end, markupAttr(segment.Attribute()))

		case text.Codeblocker:
			// Treat codeblocks the same as a monospace tag.
			// TODO: add highlighting
			appended.span(start, end, `font_family="monospace"`)

		case text.Quoteblocker:
			// TODO: pls.
			appended.span(start, end, `color="#789922"`)
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

func span(key, value string) string {
	return "<span key=\"" + value + "\">"
}
