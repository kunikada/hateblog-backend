package entry

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"hateblog/internal/domain/tag"
)

// ErrInvalidEntry signals invalid entry parameters.
var ErrInvalidEntry = errors.New("invalid entry")

// ErrInvalidListQuery signals invalid query parameters.
var ErrInvalidListQuery = errors.New("invalid list query")

// ID represents Entry identifier.
type ID = uuid.UUID

// SortType defines how entries should be ordered.
type SortType string

const (
	// SortNew orders entries by posted_at DESC.
	SortNew SortType = "new"
	// SortHot orders entries by bookmark_count DESC.
	SortHot SortType = "hot"
)

const (
	// DefaultLimit is used when ListQuery.Limit is zero.
	DefaultLimit = 20
	// MaxLimit caps ListQuery.Limit to avoid unbounded queries.
	MaxLimit = 100
)

// Entry represents a hateblog entry domain model.
type Entry struct {
	ID            ID
	URL           string
	Title         string
	Excerpt       string
	Subject       string
	BookmarkCount int
	PostedAt      time.Time
	Tags          []Tagging
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Tagging represents an attached tag with score.
type Tagging struct {
	TagID tag.ID
	Name  string
	Score int
}

// Params represents the input values required to create/update an Entry.
type Params struct {
	ID            ID
	URL           string
	Title         string
	Excerpt       string
	Subject       string
	BookmarkCount int
	PostedAt      time.Time
	Tags          []Tagging
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// New creates a new Entry after validating params.
func New(params Params) (*Entry, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}

	tags := normalizeTaggings(params.Tags)

	return &Entry{
		ID:            params.ID,
		URL:           params.URL,
		Title:         strings.TrimSpace(params.Title),
		Excerpt:       strings.TrimSpace(params.Excerpt),
		Subject:       strings.TrimSpace(params.Subject),
		BookmarkCount: params.BookmarkCount,
		PostedAt:      params.PostedAt,
		Tags:          tags,
		CreatedAt:     params.CreatedAt,
		UpdatedAt:     params.UpdatedAt,
	}, nil
}

// Update applies params to an existing Entry while validating input.
func (e *Entry) Update(params Params) error {
	if err := validateParams(params); err != nil {
		return err
	}

	tags := normalizeTaggings(params.Tags)
	e.URL = params.URL
	e.Title = strings.TrimSpace(params.Title)
	e.Excerpt = strings.TrimSpace(params.Excerpt)
	e.Subject = strings.TrimSpace(params.Subject)
	e.BookmarkCount = params.BookmarkCount
	e.PostedAt = params.PostedAt
	e.Tags = tags
	e.UpdatedAt = params.UpdatedAt
	return nil
}

func validateParams(params Params) error {
	if params.URL == "" {
		return fmt.Errorf("%w: url is required", ErrInvalidEntry)
	}
	if _, err := url.ParseRequestURI(params.URL); err != nil {
		return fmt.Errorf("%w: invalid url: %v", ErrInvalidEntry, err)
	}
	if strings.TrimSpace(params.Title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidEntry)
	}
	if params.BookmarkCount < 0 {
		return fmt.Errorf("%w: bookmark count must be >= 0", ErrInvalidEntry)
	}
	if params.PostedAt.IsZero() {
		return fmt.Errorf("%w: posted_at is required", ErrInvalidEntry)
	}
	if err := validateTaggings(params.Tags); err != nil {
		return err
	}
	return nil
}

func validateTaggings(tags []Tagging) error {
	for _, t := range tags {
		if t.TagID == uuid.Nil {
			return fmt.Errorf("%w: tag id is required", ErrInvalidEntry)
		}
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("%w: tag name is required", ErrInvalidEntry)
		}
		if t.Score < 0 || t.Score > 100 {
			return fmt.Errorf("%w: tag score must be between 0 and 100", ErrInvalidEntry)
		}
	}
	return nil
}

func normalizeTaggings(tags []Tagging) []Tagging {
	if len(tags) == 0 {
		return nil
	}
	result := make([]Tagging, len(tags))
	for i, t := range tags {
		result[i] = Tagging{
			TagID: t.TagID,
			Name:  tag.NormalizeName(t.Name),
			Score: t.Score,
		}
	}
	return result
}

// BuildSearchText builds search_text by joining non-empty fields with spaces.
func BuildSearchText(title, excerpt, url string) string {
	parts := make([]string, 0, 3)
	if trimmed := strings.TrimSpace(title); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(excerpt); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if trimmed := strings.TrimSpace(url); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " ")
}

// ListQuery represents filters applied when listing entries.
type ListQuery struct {
	Tags             []string
	MinBookmarkCount int
	Offset           int
	Limit            int
	Sort             SortType
	Keyword          string
	PostedAtFrom     time.Time
	PostedAtTo       time.Time
	MaxLimitOverride int
}

// Normalize validates and applies defaults to the query.
func (q *ListQuery) Normalize() error {
	if q.Limit <= 0 {
		q.Limit = DefaultLimit
	}
	limitCap := MaxLimit
	if q.MaxLimitOverride > 0 {
		limitCap = q.MaxLimitOverride
	}
	if q.Limit > limitCap {
		q.Limit = limitCap
	}
	if q.Offset < 0 {
		return fmt.Errorf("%w: offset must be >= 0", ErrInvalidListQuery)
	}
	if q.MinBookmarkCount < 0 {
		return fmt.Errorf("%w: min_bookmark_count must be >= 0", ErrInvalidListQuery)
	}
	q.Keyword = strings.TrimSpace(q.Keyword)
	switch q.Sort {
	case SortNew, SortHot:
		// ok
	case "":
		q.Sort = SortNew
	default:
		return fmt.Errorf("%w: unsupported sort %q", ErrInvalidListQuery, q.Sort)
	}

	if !q.PostedAtFrom.IsZero() {
		q.PostedAtFrom = q.PostedAtFrom.In(time.Local)
	}
	if !q.PostedAtTo.IsZero() {
		q.PostedAtTo = q.PostedAtTo.In(time.Local)
	}
	if !q.PostedAtFrom.IsZero() && !q.PostedAtTo.IsZero() && !q.PostedAtFrom.Before(q.PostedAtTo) {
		return fmt.Errorf("%w: posted_at_from must be before posted_at_to", ErrInvalidListQuery)
	}

	for i, name := range q.Tags {
		q.Tags[i] = tag.NormalizeName(name)
	}
	if len(q.Tags) > 1 {
		sort.Strings(q.Tags)
	}

	return nil
}
