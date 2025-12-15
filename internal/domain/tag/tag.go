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
