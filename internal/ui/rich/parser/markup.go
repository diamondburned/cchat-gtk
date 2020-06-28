package parser

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/attrmap"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/hl"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
)

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
	var appended = attrmap.NewAppendedMap()

	// Parse all segments.
	for _, segment := range content.Segments {
		start, end := segment.Bounds()

		switch segment := segment.(type) {
		case text.Linker:
			appended.Addf(start, `<a href="%s">`, html.EscapeString(segment.Link()))
			appended.Add(end, "</a>")

		case text.Imager:
			// Ends don't matter with images.
			appended.Add(start, composeImageMarkup(segment))

		case text.Colorer:
			appended.Span(start, end, fmt.Sprintf(`color="#%06X"`, segment.Color()))

		case text.Attributor:
			appended.Span(start, end, markupAttr(segment.Attribute()))

		case text.Codeblocker:
			// Syntax highlight the codeblock.
			hl.Segments(&appended, content.Content, segment)

		case text.Quoteblocker:
			// TODO: pls.
			appended.Span(start, end, `color="#789922"`)
		}
	}

	var lastIndex = 0

	for _, index := range appended.Finalize(len(content.Content)) {
		// Write the content.
		buf.WriteString(html.EscapeString(content.Content[lastIndex:index]))
		// Write the tags.
		buf.WriteString(appended.Get(index))
		// Set the last index.
		lastIndex = index
	}

	return buf.String()
}

// string constant for formatting width and height in URL fragments
const f_FragmentSize = "w=%d;h=%d"

func composeImageMarkup(imager text.Imager) string {
	u, err := url.Parse(imager.Image())
	if err != nil {
		// If the URL is invalid, then just write a normal text.
		return html.EscapeString(imager.ImageText())
	}

	// Override the URL fragment with our own.
	if w, h := imager.ImageSize(); w > 0 && h > 0 {
		u.Fragment = fmt.Sprintf(f_FragmentSize, w, h)
	}

	return fmt.Sprintf(
		`<a href="%s">%s</a>`,
		html.EscapeString(u.String()), html.EscapeString(imager.ImageText()),
	)
}

// FragmentImageSize tries to parse the width and height encoded in the URL
// fragment, which is inserted by the markup renderer. A pair of zero values are
// returned if there is none. The returned width and height will be the minimum
// of the given maxes and the encoded sizes.
func FragmentImageSize(URL string, maxw, maxh int) (w, h int) {
	u, err := url.Parse(URL)
	if err != nil {
		return
	}

	// Ignore the error, as we can check for the integers.
	fmt.Sscanf(u.Fragment, f_FragmentSize, &w, &h)

	if w > 0 && h > 0 {
		return imgutil.MaxSize(w, h, maxw, maxh)
	}

	return maxw, maxh
}

func span(key, value string) string {
	return "<span key=\"" + value + "\">"
}
