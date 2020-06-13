package humanize

import (
	"time"

	"github.com/goodsign/monday"
)

const (
	Day  = 24 * time.Hour
	Week = 7 * Day
	Year = 365 * Day
)

type truncator struct {
	d time.Duration
	s string
}

var shortTruncators = []truncator{
	{d: Day, s: "15:04"},
	{d: Week, s: "Mon 15:04"},
	{d: Year, s: "15:04 02/01"},
	{d: -1, s: "15:04 02/01/2006"},
}

func TimeAgo(t time.Time) string {
	ensureLocale()

	trunc := t
	now := time.Now()

	for _, truncator := range shortTruncators {
		trunc = trunc.Truncate(truncator.d)
		now = now.Truncate(truncator.d)

		if trunc.Equal(now) || truncator.d == -1 {
			return monday.Format(t, truncator.s, Locale)
		}
	}

	return ""
}

var longTruncators = []truncator{
	{d: Day, s: "Today at 15:04"},
	{d: Week, s: "Last Monday at 15:04"},
	{d: -1, s: "15:04 02/01/2006"},
}

func TimeAgoLong(t time.Time) string {
	ensureLocale()

	trunc := t
	now := time.Now()

	for _, truncator := range longTruncators {
		trunc = trunc.Truncate(truncator.d)
		now = now.Truncate(truncator.d)

		if trunc.Equal(now) || truncator.d == -1 {
			return monday.Format(t, truncator.s, Locale)
		}
	}

	return ""
}
