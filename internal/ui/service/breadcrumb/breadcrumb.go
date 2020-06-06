package breadcrumb

import "strings"

type Breadcrumb []string

func (b Breadcrumb) String() string {
	return strings.Join([]string(b), "/")
}

type Breadcrumber interface {
	Breadcrumb() Breadcrumb
}

// Try accepts a nilable breadcrumber and handles it appropriately.
func Try(i Breadcrumber, appended ...string) []string {
	if i == nil {
		return appended
	}
	return append(i.Breadcrumb(), appended...)
}
