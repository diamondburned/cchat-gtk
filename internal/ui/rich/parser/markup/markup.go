package markup

import (
	"bytes"
	"fmt"
	"html"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/attrmap"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/hl"
	"github.com/diamondburned/cchat/text"
	"github.com/diamondburned/imgutil"
	"github.com/gotk3/gotk3/gtk"
)

// Hyphenate controls whether or not texts should have hyphens on wrap.
var Hyphenate = false

func hyphenate(text string) string {
	if !Hyphenate {
		return text
	}
	return `<span insert_hyphens="true">` + text + `</span>`
}

// SubstringSegment slices the given rich text.
func SubstringSegment(rich text.Rich, seg text.Segment) text.Rich {
	start, end := seg.Bounds()
	substring := text.Rich{
		Content:  rich.Content[start:end],
		Segments: make([]text.Segment, 0, len(rich.Segments)),
	}

	for _, seg := range rich.Segments {
		i, j := seg.Bounds()

		// Bound-check.
		if start <= i && j <= end {
			substring.Segments = append(substring.Segments, seg)
		}
	}

	return substring
}

// RenderOutput is the output of a render.
type RenderOutput struct {
	Markup     string
	Input      string // useless to keep parts, as Go will keep all alive anyway
	Mentions   []MentionSegment
	References []ReferenceSegment
}

// MentionSegment is a type that satisfies both Segment and Mentioner.
type MentionSegment struct {
	text.Segment
	text.Mentioner
}

// ReferenceSegment is a type that satisfies both Segment and MessageReferencer.
type ReferenceSegment struct {
	text.Segment
	text.MessageReferencer
}

const (
	MentionType   = "mention"
	ReferenceType = "reference"
)

func fmtSegmentURI(stype string, ix int) string {
	u := url.URL{
		Scheme: "cchat",
		Host:   stype,
		Path:   strconv.Itoa(ix),
	}
	return u.String()
}

func (r RenderOutput) URISegment(uri string) text.Segment {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "cchat" {
		return nil
	}

	i, err := strconv.Atoi(strings.TrimPrefix(u.Path, "/"))
	if err != nil {
		panic("Invalid path " + u.Path)
	}

	switch u.Host {
	case MentionType:
		return r.Mentions[i]
	case ReferenceType:
		return r.References[i]
	default:
		panic("Unknown internal URI ID: " + u.Host)
	}
}

var simpleConfig = RenderConfig{
	NoMentionLinks: true,
	NoReferencing:  true,
}

func Render(content text.Rich) string {
	return RenderCmplxWithConfig(content, simpleConfig).Markup
}

// RenderCmplx renders content into a complete output.
func RenderCmplx(content text.Rich) RenderOutput {
	return RenderCmplxWithConfig(content, RenderConfig{})
}

type RenderConfig struct {
	// NoMentionLinks, if true, will not render any mentions.
	NoMentionLinks bool

	// NoReferencing, if true, will not parse reference links and prefer
	// mentions.
	NoReferencing bool

	// SkipImages skips rendering any image markup. This is useful for widgets
	// that already render an outside image.
	SkipImages bool

	// AnchorColor forces all anchors to be of a certain color. This is used if
	// the boolean is true. Else, all mention links will not work and regular
	// links will be of the default color.
	AnchorColor struct {
		uint32
		bool
	}
}

// setForegroundAnchor sets the AnchorColor of the render config to be that of
// the regular text foreground color.
func (c *RenderConfig) setForegroundAnchor(ctx *gtk.StyleContext) {
	rgba := ctx.GetColor(gtk.STATE_FLAG_ACTIVE)

	var color uint32
	for _, v := range rgba.Floats() { // [0.0, 1.0]
		color = (color << 8) + uint32(v*0xFF)
	}

	c.AnchorColor.bool = true
	c.AnchorColor.uint32 = text.SolidColor(color) // force alpha 100%
}

// ensureAnchorColor ensures the RenderConfig has an AnchorColor.
func (c *RenderConfig) ensureAnchorColor() {
	if c.AnchorColor.bool {
		return
	}

	styleCtx, _ := gts.App.Window.GetStyleContext()
	if styleCtx == nil {
		log.Println("Window's StyleCtx is nil.")
		return
	}

	c.setForegroundAnchor(styleCtx)
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

	// map to store mentions and references
	var mentions []MentionSegment
	var references []ReferenceSegment

	// Parse all segments.
	for _, segment := range content.Segments {
		start, end := segment.Bounds()

		// hasAnchor is used to determine if the current segment has inserted
		// any anchor tags; it is used for AnchorColor.
		var hasAnchor bool

		if linker := segment.AsLinker(); linker != nil {
			appended.Anchor(start, end, linker.Link())
			hasAnchor = true
		}

		// Only inline images if start == end per specification.
		if !cfg.SkipImages && start == end {
			if imager := segment.AsImager(); imager != nil {
				appended.Open(start, composeImageMarkup(imager))
			}

			if avatarer := segment.AsAvatarer(); avatarer != nil {
				// Ends don't matter with images.
				appended.Open(start, composeAvatarMarkup(avatarer))
			}
		}

		// Mentioner needs to be before colorer, as we'd want the below color
		// segment to also highlight the full mention as well as make the
		// padding part of the hyperlink.
		if mentioner := segment.AsMentioner(); mentioner != nil {
			// Render the mention into "cchat://mention:0" or such. Other
			// components will take care of showing the information.
			if !cfg.NoMentionLinks {
				appended.AnchorNU(start, end, fmtSegmentURI(MentionType, len(mentions)))
				hasAnchor = true
			}

			// Add the mention segment into the list regardless of hyperlinks.
			mentions = append(mentions, MentionSegment{
				Segment:   segment,
				Mentioner: mentioner,
			})
		}

		if colorer := segment.AsColorer(); colorer != nil {
			appended.Span(start, end, colorAttrs(colorer.Color(), false)...)
		} else if hasAnchor {
			cfg.ensureAnchorColor()
			appended.Span(start, end, colorAttrs(cfg.AnchorColor.uint32, false)...)
		}

		// Don't use AnchorColor for the link, as we're technically just
		// borrowing the anchor tag for its use. We should also prefer the
		// username popover (Mention) over this.
		if reference := segment.AsMessageReferencer(); reference != nil {
			if !cfg.NoReferencing && !hasAnchor {
				// Render the mention into "cchat://reference:0" or such. Other
				// components will take care of showing the information.
				appended.AnchorNU(start, end, fmtSegmentURI(ReferenceType, len(references)))
			}

			// Add the mention segment into the list regardless of hyperlinks.
			references = append(references, ReferenceSegment{
				Segment:           segment,
				MessageReferencer: reference,
			})
		}

		if attributor := segment.AsAttributor(); attributor != nil {
			appended.Span(start, end, markupAttr(attributor.Attribute()))
		}

		if codeblocker := segment.AsCodeblocker(); codeblocker != nil {
			start, end := segment.Bounds()
			// Syntax highlight the codeblock.
			hl.Segments(
				&appended,
				content.Content,
				start, end,
				codeblocker.CodeblockLanguage(),
			)
		}

		// TODO: make this not shit. Maybe make it somehow not rely on green
		// arrows. Or maybe.
		if segment.AsQuoteblocker() != nil {
			appended.Span(start, end, `color="#789922"`)
		}
	}

	var lastIndex = 0

	for _, index := range appended.Finalize(len(content.Content)) {
		// Prevent faulty backend with erroneous insertions.
		if index > len(content.Content) {
			index = len(content.Content)
		}

		// Write the content.
		buf.WriteString(html.EscapeString(content.Content[lastIndex:index]))
		// Write the tags.
		buf.WriteString(appended.Get(index))
		// Set the last index.
		lastIndex = index
	}

	return RenderOutput{
		Markup:     hyphenate(buf.String()),
		Input:      content.Content,
		Mentions:   mentions,
		References: references,
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
	hex := "#" + hexPad(rgb)

	attrs := make([]string, 1, 4)
	attrs[0] = wrapKeyValue("color", hex)

	// If we have an alpha that isn't solid (100%), then write it.
	if a < 0xFF {
		// Calculate alpha percentage.
		perc := a * 100 / 255
		attrs = append(attrs, wrapKeyValue("fgalpha", strconv.Itoa(int(perc))))
	}

	// Draw a faded background if we explicitly requested for one.
	if bg {
		// Calculate how faded the background should be for visual purposes.
		perc := a * 10 / 255 // always 10% or less.
		attrs = append(attrs, wrapKeyValue("bgalpha", strconv.Itoa(int(perc))))
		attrs = append(attrs, wrapKeyValue("bgcolor", hex))
	}

	return attrs
}

func hexPad(c uint32) string {
	hex := strconv.FormatUint(uint64(c), 16)
	if len(hex) >= 6 {
		return hex
	}
	return strings.Repeat("0", 6-len(hex)) + hex
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

func wrapKeyValue(key, value string) string {
	buf := strings.Builder{}
	buf.Grow(len(key) + len(value) + 3)
	buf.WriteString(key)
	buf.WriteByte('=')
	buf.WriteByte('"')
	buf.WriteString(value)
	buf.WriteByte('"')
	return buf.String()
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
