package duration

// TODO: put into gokit

import (
	"math"
	"strconv"
	"time"
)

func Humanize(dur time.Duration) string {
	ms := dur.Milliseconds()

	inPast := true
	if ms < 0 {
		ms *= -1
		inPast = false
	}

	milliseconds := float64(ms)
	seconds := int(math.Round(milliseconds / (1.0 * 1000.0)))
	minutes := int(math.Round(milliseconds / (60.0 * 1000.0)))
	hours := int(math.Round(milliseconds / (3600.0 * 1000.0)))
	days := int(math.Round(milliseconds / (86400.0 * 1000.0)))
	years := int(math.Round(milliseconds / (365 * 86400.0 * 1000.0)))

	formatPlural := func(num int, singular string, plural string) string {
		if num == 1 {
			return strconv.Itoa(num) + " " + singular
		} else {
			return strconv.Itoa(num) + " " + plural
		}
	}

	format := func(num int, singular string, plural string) string {
		descr := formatPlural(num, singular, plural)

		if inPast {
			return descr + " ago"
		} else {
			return "in " + descr
		}
	}

	switch {
	case years > 0:
		return format(years, "year", "years")
	case days > 0:
		return format(days, "day", "days")
	case hours > 0:
		return format(hours, "hour", "hours")
	case minutes > 0:
		return format(minutes, "minute", "minutes")
	case seconds > 0:
		return format(seconds, "second", "seconds")
	default:
		return format(int(milliseconds), "millisecond", "milliseconds")
	}
}
