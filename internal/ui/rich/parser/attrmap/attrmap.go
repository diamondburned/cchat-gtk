package attrmap

import (
	"fmt"
	"sort"
)

type AppendMap struct {
	appended map[int]string
	indices  []int
}

func NewAppendedMap() AppendMap {
	return AppendMap{
		appended: make(map[int]string),
		indices:  []int{},
	}
}

func (a *AppendMap) Span(start, end int, attr string) {
	a.Add(start, `<span `+attr+`>`)
	a.Add(end, "</span>")
}

func (a *AppendMap) Pair(start, end int, open, close string) {
	a.Add(start, open)
	a.Add(end, close)
}

func (a *AppendMap) Addf(ind int, f string, argv ...interface{}) {
	a.Add(ind, fmt.Sprintf(f, argv...))
}

func (a *AppendMap) Pad(ind int) {
	a.Add(ind, "\n")
}

func (a *AppendMap) Add(ind int, attr string) {
	if _, ok := a.appended[ind]; ok {
		a.appended[ind] += attr
		return
	}

	a.appended[ind] = attr
	a.indices = append(a.indices, ind)
}

func (a AppendMap) Get(ind int) string {
	return a.appended[ind]
}

func (a *AppendMap) Finalize(strlen int) []int {
	// make sure there's always a closing tag at the end so the entire string
	// gets flushed.
	a.Add(strlen, "")
	sort.Ints(a.indices)
	return a.indices
}
