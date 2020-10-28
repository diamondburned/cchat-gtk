// +build !nogspell

package input

import (
	"github.com/diamondburned/gspell"
	"github.com/gotk3/gotk3/gtk"
)

func init() {
	wrapSpellCheck = func(textView *gtk.TextView) {
		speller := gspell.GetFromGtkTextView(textView)
		speller.BasicSetup()
	}
}
