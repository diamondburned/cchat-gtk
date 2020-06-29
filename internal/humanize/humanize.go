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
	return timeAgo(t, shortTruncators)
}

var longTruncators = []truncator{
	{d: Day, s: "Today at 15:04"},
	{d: Week, s: "Last Monday at 15:04"},
	{d: -1, s: "15:04 02/01/2006"},
}

func TimeAgoLong(t time.Time) string {
	return timeAgo(t, longTruncators)
}

func TimeAgoShort(t time.Time) string {
	t = t.Local()
	return monday.Format(t, "15:04", Locale)
}

func timeAgo(t time.Time, truncs []truncator) string {
	t = t.Local()
	ensureLocale()

	trunc := t
	now := time.Now().Local()

	for _, truncator := range truncs {
		trunc = trunc.Truncate(truncator.d)
		now = now.Truncate(truncator.d)

		if trunc.Equal(now) || truncator.d == -1 {
			return monday.Format(t, truncator.s, Locale)
		}
	}

	return ""
}
