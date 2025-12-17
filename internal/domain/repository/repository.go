package repository

import (
	"context"
	"time"

	"hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"
)

//go:generate mockgen -destination=./mocks_entry_repository.go -package=repository hateblog/internal/domain/repository EntryRepository

// EntryRepository defines storage operations for entries.
type EntryRepository interface {
	Get(ctx context.Context, id entry.ID) (*entry.Entry, error)
	List(ctx context.Context, query entry.ListQuery) ([]*entry.Entry, error)
	Count(ctx context.Context, query entry.ListQuery) (int64, error)
	Create(ctx context.Context, entry *entry.Entry) error
	Update(ctx context.Context, entry *entry.Entry) error
	Delete(ctx context.Context, id entry.ID) error
	ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]ArchiveCount, error)
}

//go:generate mockgen -destination=./mocks_tag_repository.go -package=repository hateblog/internal/domain/repository TagRepository

// TagRepository defines storage operations for tags.
type TagRepository interface {
	Get(ctx context.Context, id tag.ID) (*tag.Tag, error)
	GetByName(ctx context.Context, name string) (*tag.Tag, error)
	List(ctx context.Context, limit, offset int) ([]tag.Tag, error)
	Upsert(ctx context.Context, tag *tag.Tag) error
	Delete(ctx context.Context, id tag.ID) error
	IncrementViewHistory(ctx context.Context, tagID tag.ID, viewedAt time.Time) error
}

// SearchHistoryRepository stores aggregated search metrics.
type SearchHistoryRepository interface {
	Record(ctx context.Context, query string, searchedAt time.Time) error
}

// ClickMetricsRepository stores click counts per entry/date.
type ClickMetricsRepository interface {
	Increment(ctx context.Context, entryID entry.ID, clickedAt time.Time) error
}

// ArchiveCount represents aggregated entry counts per day.
type ArchiveCount struct {
	Date  time.Time
	Count int
}
