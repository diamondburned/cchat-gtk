package attrmap

import (
	"fmt"
	"sort"
	"strings"
)

type AppendMap struct {
	appended  map[int]string // for opening tags
	prepended map[int]string // for closing tags
	indices   []int
}

func NewAppendedMap() AppendMap {
	return AppendMap{
		appended:  map[int]string{},
		prepended: map[int]string{},
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

func (a *AppendMap) Span(start, end int, attrs ...string) {
	a.Openf(start, "<span %s>", strings.Join(attrs, " "))
	a.Close(end, "</span>")
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
		a.appended[ind] = str + attr // append
		return
	}

	a.appended[ind] = attr
	a.appendIndex(ind)
}

func (a *AppendMap) Close(ind int, attr string) {
	if str, ok := a.prepended[ind]; ok {
		a.prepended[ind] = attr + str // prepend
		return
	}

	a.prepended[ind] = attr
	a.appendIndex(ind)
}

func (a AppendMap) Get(ind int) (tags string) {
	if t, ok := a.appended[ind]; ok {
		tags += t
	}
	if t, ok := a.prepended[ind]; ok {
		tags += t
	}
	return
}

func (a *AppendMap) Finalize(strlen int) []int {
	// make sure there's always a closing tag at the end so the entire string
	// gets flushed.
	a.Close(strlen, "")
	sort.Ints(a.indices)
	return a.indices
}
