package rich

import (
	"html"

	"github.com/diamondburned/cchat/text"
)

func Small(text string) string {
	return `<span size="small" color="#808080">` + text + "</span>"
}

func MakeRed(content text.Rich) string {
	return `<span color="red">` + html.EscapeString(content.Content) + `</span>`
}

// used for grabbing text without changing state
type nullLabel struct {
	text.Rich
}

func (n *nullLabel) SetLabel(t text.Rich) { n.Rich = t }
