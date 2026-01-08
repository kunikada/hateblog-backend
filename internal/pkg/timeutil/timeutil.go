package timeutil

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	locationMu sync.RWMutex
	location   = time.UTC
)

// SetLocation sets the default application timezone.
func SetLocation(name string) error {
	tz := strings.TrimSpace(name)
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("load location %q: %w", tz, err)
	}
	locationMu.Lock()
	location = loc
	locationMu.Unlock()
	return nil
}

// Location returns the configured timezone.
func Location() *time.Location {
	locationMu.RLock()
	loc := location
	locationMu.RUnlock()
	return loc
}

// Now returns the current time in the configured timezone.
func Now() time.Time {
	return time.Now().In(Location())
}

// InLocation converts the given time to the configured timezone.
func InLocation(t time.Time) time.Time {
	return t.In(Location())
}

// DateInLocation returns midnight UTC for the date in the configured timezone.
func DateInLocation(t time.Time) time.Time {
	local := t.In(Location())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}
