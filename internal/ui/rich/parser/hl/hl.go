package hl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/diamondburned/cchat-gtk/internal/ui/config"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/attrmap"
)

var (
	lexerMap = map[string]chroma.Lexer{}

	// this is not thread-safe, but we can reuse everything, as we know Gtk will
	// run everything in the main thread.
	fmtter = formatter{}

	// tokenType -> span attrs
	css = map[chroma.TokenType]string{}
)

func init() {
	var name = "algol_nu" // default
	ChangeStyle(name)
	config.AppearanceAdd("Code Highlight Style", config.InputEntry(&name, ChangeStyle))
}

func Tokenize(language, source string) chroma.Iterator {
	var lexer = getLexer(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	i, _ := lexer.Tokenise(nil, source)
	return i
}

func Segments(appendmap *attrmap.AppendMap, src string, start, end int, lang string) {
	appendmap.Span(
		start, end,
		`font_family="monospace"`,
		`insert_hyphens="false"`, // all my homies hate hyphens
	)

	if i := Tokenize(lang, src[start:end]); i != nil {
		fmtter.segments(appendmap, start, i)
	}
}

func ChangeStyle(styleName string) error {
	s := styles.Get(styleName)

	// styleName == "" => no highlighting, not an error
	if s == styles.Fallback && styleName != "" {
		return errors.New("Unknown style")
	}

	css = styleToCSS(s)
	return nil
}

func getLexer(lang string) chroma.Lexer {
	v, ok := lexerMap[lang]
	if ok {
		return v
	}

	v = lexers.Get(lang)
	if v != nil {
		lexerMap[lang] = v
		return v
	}

	return nil
}

// Formatter that generates Pango markup.
type formatter struct {
	highlightRanges [][2]int
}

func (f *formatter) reset() {
	f.highlightRanges = f.highlightRanges[:0]
}

func (f *formatter) segments(appendmap *attrmap.AppendMap, offset int, iter chroma.Iterator) {
	f.reset()

	for _, token := range iter.Tokens() {
		attr := f.styleAttr(token.Type)

		if attr != "" {
			appendmap.Openf(offset, `<span %s>`, attr)
		}

		offset += len(token.Value)

		if attr != "" {
			appendmap.Close(offset, "</span>")
		}
	}
}

func (f *formatter) styleAttr(tt chroma.TokenType) string {
	if _, ok := css[tt]; !ok {
		tt = tt.SubCategory()
	}
	if _, ok := css[tt]; !ok {
		tt = tt.Category()
	}
	if t, ok := css[tt]; ok {
		return t
	}

	return ""
}

func styleToCSS(style *chroma.Style) map[chroma.TokenType]string {
	classes := map[chroma.TokenType]string{}
	bg := style.Get(chroma.Background)

	for t := range chroma.StandardTypes {
		var entry = style.Get(t)
		if t != chroma.Background {
			entry = entry.Sub(bg)
		}
		if entry.IsZero() {
			continue
		}
		classes[t] = styleEntryToTag(entry)
	}
	return classes
}

func styleEntryToTag(e chroma.StyleEntry) string {
	var attrs = make([]string, 0, 1)

	if e.Colour.IsSet() {
		attrs = append(attrs, fmt.Sprintf(`foreground="%s"`, e.Colour.String()))
	}
	if e.Bold == chroma.Yes {
		attrs = append(attrs, `weight="bold"`)
	}
	if e.Italic == chroma.Yes {
		attrs = append(attrs, `style="italic"`)
	}
	if e.Underline == chroma.Yes {
		attrs = append(attrs, `underline="single"`)
	}

	return strings.Join(attrs, " ")
}
