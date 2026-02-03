package apptime

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}
	time.Local = loc
	os.Exit(m.Run())
}

func TestTruncateToDay(t *testing.T) {
	t.Run("UTC input crossing JST date boundary", func(t *testing.T) {
		// UTC 2024-01-01 23:30 = JST 2024-01-02 08:30
		utc := time.Date(2024, 1, 1, 23, 30, 0, 0, time.UTC)
		got := TruncateToDay(utc)
		require.Equal(t, 2024, got.Year())
		require.Equal(t, time.January, got.Month())
		require.Equal(t, 2, got.Day())
		require.Equal(t, 0, got.Hour())
		require.Equal(t, 0, got.Minute())
		require.Equal(t, 0, got.Second())
		require.Equal(t, time.Local, got.Location())
	})

	t.Run("already local timezone", func(t *testing.T) {
		jst := time.Date(2024, 6, 15, 14, 30, 45, 0, time.Local)
		got := TruncateToDay(jst)
		require.Equal(t, 2024, got.Year())
		require.Equal(t, time.June, got.Month())
		require.Equal(t, 15, got.Day())
		require.Equal(t, 0, got.Hour())
	})

	t.Run("midnight stays same day", func(t *testing.T) {
		jst := time.Date(2024, 3, 1, 0, 0, 0, 0, time.Local)
		got := TruncateToDay(jst)
		require.Equal(t, 1, got.Day())
		require.Equal(t, time.March, got.Month())
	})
}

func TestParseDate(t *testing.T) {
	t.Run("valid date", func(t *testing.T) {
		got, err := ParseDate("20240101")
		require.NoError(t, err)
		require.Equal(t, 2024, got.Year())
		require.Equal(t, time.January, got.Month())
		require.Equal(t, 1, got.Day())
		require.Equal(t, 0, got.Hour())
		require.Equal(t, time.Local, got.Location())
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := ParseDate("2024-01-01")
		require.Error(t, err)
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := ParseDate("")
		require.Error(t, err)
	})
}

func TestDayRange(t *testing.T) {
	t.Run("normal day", func(t *testing.T) {
		start, end, err := DayRange("20240315")
		require.NoError(t, err)
		require.Equal(t, 15, start.Day())
		require.Equal(t, 16, end.Day())
		require.Equal(t, time.Local, start.Location())
	})

	t.Run("end of month", func(t *testing.T) {
		start, end, err := DayRange("20240131")
		require.NoError(t, err)
		require.Equal(t, 31, start.Day())
		require.Equal(t, time.February, end.Month())
		require.Equal(t, 1, end.Day())
	})

	t.Run("invalid date", func(t *testing.T) {
		_, _, err := DayRange("invalid")
		require.Error(t, err)
	})
}

func TestYearRange(t *testing.T) {
	t.Run("normal year", func(t *testing.T) {
		start, end, err := YearRange(2024)
		require.NoError(t, err)
		require.Equal(t, 2024, start.Year())
		require.Equal(t, time.January, start.Month())
		require.Equal(t, 1, start.Day())
		require.Equal(t, 2025, end.Year())
		require.Equal(t, time.January, end.Month())
		require.Equal(t, time.Local, start.Location())
	})

	t.Run("year too small", func(t *testing.T) {
		_, _, err := YearRange(1999)
		require.Error(t, err)
	})

	t.Run("year too large", func(t *testing.T) {
		_, _, err := YearRange(10000)
		require.Error(t, err)
	})
}

func TestMonthRange(t *testing.T) {
	t.Run("normal month", func(t *testing.T) {
		start, end, err := MonthRange(2024, 2)
		require.NoError(t, err)
		require.Equal(t, time.February, start.Month())
		require.Equal(t, 1, start.Day())
		require.Equal(t, time.March, end.Month())
		require.Equal(t, 1, end.Day())
		require.Equal(t, time.Local, start.Location())
	})

	t.Run("december wraps to next year", func(t *testing.T) {
		start, end, err := MonthRange(2024, 12)
		require.NoError(t, err)
		require.Equal(t, time.December, start.Month())
		require.Equal(t, 2025, end.Year())
		require.Equal(t, time.January, end.Month())
	})

	t.Run("month zero", func(t *testing.T) {
		_, _, err := MonthRange(2024, 0)
		require.Error(t, err)
	})

	t.Run("month 13", func(t *testing.T) {
		_, _, err := MonthRange(2024, 13)
		require.Error(t, err)
	})
}

func TestISOWeekRange(t *testing.T) {
	t.Run("week 1 of 2024", func(t *testing.T) {
		start, end, err := ISOWeekRange(2024, 1)
		require.NoError(t, err)
		require.Equal(t, time.Monday, start.Weekday())
		require.Equal(t, 7, int(end.Sub(start).Hours()/24))
		require.Equal(t, time.Local, start.Location())
		isoYear, isoWeek := start.ISOWeek()
		require.Equal(t, 2024, isoYear)
		require.Equal(t, 1, isoWeek)
	})

	t.Run("week zero", func(t *testing.T) {
		_, _, err := ISOWeekRange(2024, 0)
		require.Error(t, err)
	})

	t.Run("week 54", func(t *testing.T) {
		_, _, err := ISOWeekRange(2024, 54)
		require.Error(t, err)
	})
}
