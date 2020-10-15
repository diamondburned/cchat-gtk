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
	Input    string // useless to keep parts, as Go will keep all alive anyway
	Mentions []MentionSegment
}

// MentionSegment is a type that satisfies both Segment and Mentioner.
type MentionSegment struct {
	text.Segment
	text.Mentioner
}

// f_Mention is used to print and parse mention URIs.
const f_Mention = "cchat://mention/%d" // %d == Mentions[i]

// IsMention returns the mention if the URI is correct, or nil if none.
func (r RenderOutput) IsMention(uri string) text.Segment {
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
	return RenderCmplxWithConfig(content, RenderConfig{})
}

type RenderConfig struct {
	// NoMentionLinks prevents the renderer from wrapping mentions with a
	// hyperlink. This prevents invalid colors.
	NoMentionLinks bool
}

// NoMentionLinks is the config to render author names. It disables author
// mention links, as there's no way to make normal names not appear blue.
var NoMentionLinks = RenderConfig{
	NoMentionLinks: true,
}

func RenderCmplxWithConfig(content text.Rich, cfg RenderConfig) RenderOutput {
	// Fast path.
	if len(content.Segments) == 0 {
		return RenderOutput{
			Markup: hyphenate(html.EscapeString(content.Content)),
			Input:  content.Content,
		}
	}

	buf := bytes.Buffer{}
	buf.Grow(len(content.Content))

	// Sort so that all ending points are sorted decrementally. We probably
	// don't need SliceStable here, as we're sorting again.
	sort.Slice(content.Segments, func(i, j int) bool {
		_, i = content.Segments[i].Bounds()
		_, j = content.Segments[j].Bounds()
		return i > j
	})

	// Sort so that all starting points are sorted incrementally.
	sort.SliceStable(content.Segments, func(i, j int) bool {
		i, _ = content.Segments[i].Bounds()
		j, _ = content.Segments[j].Bounds()
		return i < j
	})

	// map to append strings to indices
	var appended = attrmap.NewAppendedMap()

	// map to store mentions
	var mentions []MentionSegment

	// Parse all segments.
	for _, segment := range content.Segments {
		start, end := segment.Bounds()

		if linker := segment.AsLinker(); linker != nil {
			appended.Anchor(start, end, linker.Link())
		}

		// Only inline images if start == end per specification.
		if start == end {
			if imager := segment.AsImager(); imager != nil {
				appended.Open(start, composeImageMarkup(imager))
			}

			if avatarer := segment.AsAvatarer(); avatarer != nil {
				// Ends don't matter with images.
				appended.Open(start, composeAvatarMarkup(avatarer))
			}
		}

		if colorer := segment.AsColorer(); colorer != nil {
			appended.Span(start, end, colorAttrs(colorer.Color(), false)...)
		}

		// Mentioner needs to be before colorer, as we'd want the below color
		// segment to also highlight the full mention as well as make the
		// padding part of the hyperlink.
		if mentioner := segment.AsMentioner(); mentioner != nil {
			// Render the mention into "cchat://mention:0" or such. Other
			// components will take care of showing the information.
			if !cfg.NoMentionLinks {
				appended.AnchorNU(start, end, fmt.Sprintf(f_Mention, len(mentions)))
			}

			// Add the mention segment into the list regardless of hyperlinks.
			mentions = append(mentions, MentionSegment{
				Segment:   segment,
				Mentioner: mentioner,
			})

			if colorer := segment.AsColorer(); colorer != nil {
				// Only pad the name and add a dimmed background if the bounds
				// do not cover the whole segment.
				var cover = (start == 0) && (end == len(content.Content))
				appended.Span(start, end, colorAttrs(colorer.Color(), !cover)...)
				if !cover {
					appended.Pad(start, end)
				}
			}
		}

		if attributor := segment.AsAttributor(); attributor != nil {
			appended.Span(start, end, markupAttr(attributor.Attribute()))
		}

		if codeblocker := segment.AsCodeblocker(); codeblocker != nil {
			start, end := segment.Bounds()
			// Syntax highlight the codeblock.
			hl.Segments(&appended, content.Content, start, end, codeblocker.CodeblockLanguage())
		}

		// TODO: make this not shit. Maybe make it somehow not rely on green
		// arrows. Or maybe.
		if segment.AsQuoteblocker() != nil {
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
		Input:    content.Content,
		Mentions: mentions,
	}
}

// splitRGBA splits the given rgba integer into rgb and a.
func splitRGBA(rgba uint32) (rgb, a uint32) {
	rgb = rgba >> 8 // extract the RGB bits
	a = rgba & 0xFF // extract the A bits
	return
}

// colorAttrs renders the given color into a list of attributes.
func colorAttrs(c uint32, bg bool) []string {
	// Split the RGBA color value to calculate.
	rgb, a := splitRGBA(c)

	// Render the hex representation beforehand.
	hex := fmt.Sprintf("#%06X", rgb)

	attrs := make([]string, 1, 4)
	attrs[0] = fmt.Sprintf(`color="%s"`, hex)

	// If we have an alpha that isn't solid (100%), then write it.
	if a < 0xFF {
		// Calculate alpha percentage.
		perc := a * 100 / 255
		attrs = append(attrs, fmt.Sprintf(`fgalpha="%d%%"`, perc))
	}

	// Draw a faded background if we explicitly requested for one.
	if bg {
		// Calculate how faded the background should be for visual purposes.
		perc := a * 10 / 255 // always 10% or less.
		attrs = append(attrs, fmt.Sprintf(`bgalpha="%d%%"`, perc))
		attrs = append(attrs, fmt.Sprintf(`bgcolor="%s"`, hex))
	}

	return attrs
}

const (
	// string constant for formatting width and height in URL fragments
	f_FragmentSize      = "w=%d;h=%d"
	f_AnchorNoUnderline = `<a href="%s"><span underline="none">%s</span></a>`
)

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
		f_AnchorNoUnderline,
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
		f_AnchorNoUnderline,
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
	if attr.Has(text.AttributeBold) {
		attrs = append(attrs, `weight="bold"`)
	}
	if attr.Has(text.AttributeItalics) {
		attrs = append(attrs, `style="italic"`)
	}
	if attr.Has(text.AttributeUnderline) {
		attrs = append(attrs, `underline="single"`)
	}
	if attr.Has(text.AttributeStrikethrough) {
		attrs = append(attrs, `strikethrough="true"`)
	}
	if attr.Has(text.AttributeSpoiler) {
		attrs = append(attrs, `alpha="35%"`) // no fancy click here
	}
	if attr.Has(text.AttributeMonospace) {
		attrs = append(attrs, `font_family="monospace"`)
	}
	if attr.Has(text.AttributeDimmed) {
		attrs = append(attrs, `alpha="35%"`)
	}

	return strings.Join(attrs, " ")
}
