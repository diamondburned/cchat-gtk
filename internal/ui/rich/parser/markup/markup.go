package markup

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/attrmap"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/hl"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
)

// Hyphenate controls whether or not texts should have hyphens on wrap.
var Hyphenate = false

func hyphenate(text string) string {
	return fmt.Sprintf(`<span insert_hyphens="%t">%s</span>`, Hyphenate, text)
}

// RenderOutput is the output of a render.
type RenderOutput struct {
	Markup   string
	Mentions []text.Mentioner
}

// f_Mention is used to print and parse mention URIs.
const f_Mention = "cchat://mention:%d" // %d == Mentions[i]

// IsMention returns the mention if the URI is correct, or nil if none.
func (r RenderOutput) IsMention(uri string) text.Mentioner {
	var i int

	if _, err := fmt.Sscanf(uri, f_Mention, &i); err != nil {
		return nil
	}

	if i >= len(r.Mentions) {
		return nil
	}

	return r.Mentions[i]
}

func Render(content text.Rich) string {
	return RenderCmplx(content).Markup
}

// RenderCmplx renders content into a complete output.
func RenderCmplx(content text.Rich) RenderOutput {
	// Fast path.
	if len(content.Segments) == 0 {
		return RenderOutput{
			Markup: hyphenate(html.EscapeString(content.Content)),
		}
	}

	buf := bytes.Buffer{}
	buf.Grow(len(content.Content))

	// Sort so that all starting points are sorted incrementally.
	sort.SliceStable(content.Segments, func(i, j int) bool {
		i, _ = content.Segments[i].Bounds()
		j, _ = content.Segments[j].Bounds()
		return i < j
	})

	// map to append strings to indices
	var appended = attrmap.NewAppendedMap()

	// map to store mentions
	var mentions []text.Mentioner

	// Parse all segments.
	for _, segment := range content.Segments {
		start, end := segment.Bounds()

		if segment, ok := segment.(text.Linker); ok {
			appended.Openf(start, `<a href="%s">`, html.EscapeString(segment.Link()))
			appended.Close(end, "</a>")
		}

		if segment, ok := segment.(text.Imager); ok {
			// Ends don't matter with images.
			appended.Open(start, composeImageMarkup(segment))
		}

		if segment, ok := segment.(text.Avatarer); ok {
			// Ends don't matter with images.
			appended.Open(start, composeAvatarMarkup(segment))
		}

		// Mentioner needs to be before colorer, as we'd want the below color
		// segment to also highlight the full mention as well as make the
		// padding part of the hyperlink.
		if segment, ok := segment.(text.Mentioner); ok {
			// Render the mention into "cchat://mention:0" or such. Other
			// components will take care of showing the information.
			appended.AnchorNU(start, end, fmt.Sprintf(f_Mention, len(mentions)))
			mentions = append(mentions, segment)
		}

		if segment, ok := segment.(text.Colorer); ok {
			var covered = attrmap.CoverAll(content, start, end)
			appended.Span(start, end, color(segment.Color(), !covered)...)
			if !covered { // add padding if doesn't cover all
				appended.Pad(start, end)
			}
		}

		if segment, ok := segment.(text.Attributor); ok {
			appended.Span(start, end, markupAttr(segment.Attribute()))
		}

		if segment, ok := segment.(text.Codeblocker); ok {
			// Syntax highlight the codeblock.
			hl.Segments(&appended, content.Content, segment)
		}

		// TODO: make this not shit. Maybe make it somehow not rely on green
		// arrows. Or maybe.
		if _, ok := segment.(text.Quoteblocker); ok {
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

	return RenderOutput{
		Markup:   hyphenate(buf.String()),
		Mentions: mentions,
	}
}

func color(c uint32, bg bool) []string {
	var hex = fmt.Sprintf("#%06X", c)

	var attrs = []string{
		fmt.Sprintf(`color="%s"`, hex),
	}

	if bg {
		attrs = append(
			attrs,
			`bgalpha="10%"`,
			fmt.Sprintf(`bgcolor="%s"`, hex),
		)
	}

	return attrs
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

func composeAvatarMarkup(avatarer text.Avatarer) string {
	u, err := url.Parse(avatarer.Avatar())
	if err != nil {
		// If the URL is invalid, then just write a normal text.
		return html.EscapeString(avatarer.AvatarText())
	}

	// Override the URL fragment with our own.
	if size := avatarer.AvatarSize(); size > 0 {
		u.Fragment = fmt.Sprintf(f_FragmentSize, size, size) + ";round"
	}

	return fmt.Sprintf(
		`<a href="%s">%s</a>`,
		html.EscapeString(u.String()), html.EscapeString(avatarer.AvatarText()),
	)
}

// FragmentImageSize tries to parse the width and height encoded in the URL
// fragment, which is inserted by the markup renderer. A pair of zero values are
// returned if there is none. The returned width and height will be the minimum
// of the given maxes and the encoded sizes.
func FragmentImageSize(URL string, maxw, maxh int) (w, h int, round bool) {
	u, err := url.Parse(URL)
	if err != nil {
		return
	}

	// Ignore the error, as we can check for the integers.
	fmt.Sscanf(u.Fragment, f_FragmentSize, &w, &h)
	round = strings.HasSuffix(u.Fragment, ";round")

	if w > 0 && h > 0 {
		w, h = imgutil.MaxSize(w, h, maxw, maxh)
		return
	}

	return maxw, maxh, round
}

func span(key, value string) string {
	return "<span key=\"" + value + "\">"
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
		attrs = append(attrs, `alpha="35%"`) // no fancy click here
	}
	if attr.Has(text.AttrMonospace) {
		attrs = append(attrs, `font_family="monospace"`)
	}
	if attr.Has(text.AttrDimmed) {
		attrs = append(attrs, `alpha="35%"`)
	}

	return strings.Join(attrs, " ")
}
