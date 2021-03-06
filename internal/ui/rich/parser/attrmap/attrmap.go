package attrmap

import (
	"bytes"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/diamondburned/cchat/text"
)

type AppendMap struct {
	appended  map[int][]byte // for opening tags
	prepended map[int][]byte // for closing tags
	indices   []int
}

func NewAppendedMap() AppendMap {
	return AppendMap{
		appended:  map[int][]byte{},
		prepended: map[int][]byte{},
		indices:   []int{},
	}
}

func (a *AppendMap) appendIndex(ind int) {
	// Backwards search which should make things faster.
	for i := len(a.indices) - 1; i >= 0; i-- {
		if a.indices[i] == ind {
			return
		}
	}

	a.indices = append(a.indices, ind)
}

func (a *AppendMap) Anchor(start, end int, href string) {
	a.AnchorNU(start, end, href)
}

// AnchorNU makes a new <a> tag without underlines and colors.
func (a *AppendMap) AnchorNU(start, end int, href string) {
	a.Openf(start, `<a href="`+html.EscapeString(href)+`">`)
	a.Close(end, "</a>")
	// a.Anchor(start, end, href)
	a.Span(start, end, `underline="none"`)
}

func (a *AppendMap) Span(start, end int, attrs ...string) {
	a.Open(start, "<span "+strings.Join(attrs, " ")+">")
	a.Close(end, "</span>")
}

// Pad inserts 2 spaces into start and end. It ensures that not more than 1
// space is inserted.
func (a *AppendMap) Pad(start, end int) {
	// Ensure that the starting point does not already have a space.
	if !posHaveSpace(a.appended, start) {
		a.Open(start, " ")
	}
	if !posHaveSpace(a.prepended, end) {
		a.Close(end, " ")
	}
}

func posHaveSpace(tags map[int][]byte, pos int) bool {
	tg, ok := tags[pos]
	if !ok || len(tg) == 0 {
		return false
	}

	// Check trailing spaces.
	if tg[0] == ' ' {
		return true
	}
	if tg[len(tg)-1] == ' ' {
		return true
	}

	// Check spaces mid-tag. This works because strings are always escaped.
	return bytes.Contains(tg, []byte("> <"))
}

func (a *AppendMap) Pair(start, end int, open, close string) {
	a.Open(start, open)
	a.Close(end, close)
}

func (a *AppendMap) Openf(ind int, f string, argv ...interface{}) {
	a.Open(ind, fmt.Sprintf(f, argv...))
}

func (a *AppendMap) Open(ind int, attr string) {
	if str, ok := a.appended[ind]; ok {
		a.appended[ind] = append(str, []byte(attr)...) // append
		return
	}

	a.appended[ind] = []byte(attr)
	a.appendIndex(ind)
}

func (a *AppendMap) Close(ind int, attr string) {
	if str, ok := a.prepended[ind]; ok {
		a.prepended[ind] = append([]byte(attr), str...) // prepend
		return
	}

	a.prepended[ind] = []byte(attr)
	a.appendIndex(ind)
}

func (a AppendMap) Get(ind int) (tags string) {
	appended := a.appended[ind]
	prepended := a.prepended[ind]

	// Borrowing appended's backing array to add prepended is probably fine, as
	// the length of the actual appended slice is going to stay the same.
	return string(append(prepended, appended...))
}

func (a *AppendMap) Finalize(strlen int) []int {
	// make sure there's always a closing tag at the end so the entire string
	// gets flushed.
	a.Close(strlen, "")
	sort.Ints(a.indices)
	return a.indices
}

// CoverAll returns true if the given start and end covers the entire text
// segment.
func CoverAll(txt text.Rich, start, end int) bool {
	return start == 0 && end == len(txt.Content)
}
