package repository

import (
	"context"

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
}

//go:generate mockgen -destination=./mocks_tag_repository.go -package=repository hateblog/internal/domain/repository TagRepository

// TagRepository defines storage operations for tags.
type TagRepository interface {
	Get(ctx context.Context, id tag.ID) (*tag.Tag, error)
	GetByName(ctx context.Context, name string) (*tag.Tag, error)
	List(ctx context.Context, limit, offset int) ([]tag.Tag, error)
	Upsert(ctx context.Context, tag *tag.Tag) error
	Delete(ctx context.Context, id tag.ID) error
}
