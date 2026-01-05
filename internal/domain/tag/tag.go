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

// NormalizeName trims spaces and converts the tag name to lower-case.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
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
