package apptime

import (
	"fmt"
	"time"
)

const createdAtFallbackThreshold = 24 * time.Hour

// Now returns current time.
// It keeps monotonic clock readings for elapsed-time measurements.
func Now() time.Time {
	return time.Now()
}

// TruncateToDay returns midnight (00:00:00) of the given time in the
// application timezone (time.Local).
func TruncateToDay(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// ResolveCreatedAt returns created_at from now/posted_at rule.
// When posted_at is 24 hours or older, posted_at is used as created_at.
func ResolveCreatedAt(now, postedAt time.Time) time.Time {
	if now.Sub(postedAt) >= createdAtFallbackThreshold {
		return postedAt
	}
	return now
}

// ParseDate parses a "YYYYMMDD" date string in the application timezone.
func ParseDate(date string) (time.Time, error) {
	t, err := time.ParseInLocation("20060102", date, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date: %s", date)
	}
	return t, nil
}

// DayRange returns the start and end of a day parsed from a "YYYYMMDD" string.
// start is midnight, end is midnight of the following day.
func DayRange(date string) (start, end time.Time, err error) {
	start, err = ParseDate(date)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.AddDate(0, 0, 1), nil
}

// YearRange returns the start and end of the given year.
// start is January 1 00:00, end is January 1 00:00 of the following year.
func YearRange(year int) (start, end time.Time, err error) {
	if year < 2000 || year > 9999 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid year: %d", year)
	}
	start = time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(1, 0, 0), nil
}

// MonthRange returns the start and end of the given month.
// start is the 1st at 00:00, end is the 1st of the following month at 00:00.
func MonthRange(year, month int) (start, end time.Time, err error) {
	if month < 1 || month > 12 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month: %d", month)
	}
	start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(0, 1, 0), nil
}

// ISOWeekRange returns the Monday-to-Monday range for the given ISO week.
// start is Monday 00:00, end is the following Monday 00:00.
func ISOWeekRange(year, week int) (start, end time.Time, err error) {
	if week < 1 || week > 53 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid week: %d", week)
	}
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start = jan4.AddDate(0, 0, -(weekday - 1))
	start = start.AddDate(0, 0, (week-1)*7)
	isoYear, isoWeek := start.ISOWeek()
	if isoYear != year || isoWeek != week {
		return time.Time{}, time.Time{}, fmt.Errorf("week %d out of range for year %d", week, year)
	}
	return start, start.AddDate(0, 0, 7), nil
}
