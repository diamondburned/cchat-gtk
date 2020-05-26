package humanize

import (
	"log"
	"strings"
	"sync"
	"unicode"

	"github.com/Xuanwo/go-locale"
	"github.com/goodsign/monday"
)

var Locale monday.Locale = monday.LocaleEnUS // changed on init

var localeOnce sync.Once

func lettersOnly(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1
	}, str)
}

func ensureLocale() {
	localeOnce.Do(func() {
		if tag, err := locale.Detect(); err == nil {
			Locale = monday.Locale(lettersOnly(tag.String()))
		}

		// Check if locale is supported
		for _, locale := range monday.ListLocales() {
			if lettersOnly(string(locale)) == string(Locale) {
				return
			}
		}

		log.Println("Locale", Locale, "not found, defaulting to en_US")
		Locale = monday.LocaleEnUS
	})
}
