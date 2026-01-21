package tag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ErrInvalidTag signals invalid tag parameters.
var ErrInvalidTag = errors.New("invalid tag")

// ID represents tag identifier.
type ID = uuid.UUID

// Tag represents an entry tag.
type Tag struct {
	ID   ID
	Name string
}

// NormalizeName trims spaces, converts the tag name to lower-case, and truncates to 255 bytes.
func NormalizeName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	// Truncate to 255 bytes to match DB constraint
	// Must ensure we don't cut in the middle of a UTF-8 multibyte character
	if len(normalized) > 255 {
		// Find the last valid rune boundary within 255 bytes
		normalized = truncateUTF8(normalized, 255)
	}
	return normalized
}

// truncateUTF8 truncates a string to maxBytes without breaking UTF-8 encoding
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// Walk backwards to find a valid UTF-8 boundary
	for i := maxBytes; i > 0; i-- {
		if (s[i] & 0xC0) != 0x80 {
			// This is not a continuation byte (10xxxxxx), so it's a valid boundary
			return s[:i]
		}
	}
	return ""
}

// New creates a new Tag entity.
func New(id ID, name string) (Tag, error) {
	norm := NormalizeName(name)
	if norm == "" {
		return Tag{}, fmt.Errorf("%w: name is required", ErrInvalidTag)
	}
	return Tag{
		ID:   id,
		Name: norm,
	}, nil
}

// TrendingTag represents a tag with its occurrence count in recent entries.
type TrendingTag struct {
	ID              ID
	Name            string
	OccurrenceCount int // Number of entries with this tag in the specified period
	EntryCount      int // Total number of entries with this tag
}

// ClickedTag represents a tag with its click count from recent entries.
type ClickedTag struct {
	ID         ID
	Name       string
	ClickCount int // Number of clicks on entries with this tag in the specified period
	EntryCount int // Total number of entries with this tag
}
