package entry

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	params := Params{
		ID:            uuid.New(),
		URL:           "https://example.com/article",
		Title:         " Example ",
		Excerpt:       " excerpt ",
		Subject:       "subject",
		BookmarkCount: 12,
		PostedAt:      time.Now(),
		Tags: []Tagging{{
			TagID: uuid.New(),
			Name:  "Go",
			Score: 0.9,
		}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	entry, err := New(params)
	require.NoError(t, err)
	require.Equal(t, "Example", entry.Title)
	require.Equal(t, 12, entry.BookmarkCount)
	require.Len(t, entry.Tags, 1)
}

func TestNew_Invalid(t *testing.T) {
	tests := map[string]Params{
		"empty url": {
			Title:         "title",
			BookmarkCount: 1,
			PostedAt:      time.Now(),
		},
		"invalid url": {
			URL:           "://invalid",
			Title:         "title",
			BookmarkCount: 1,
			PostedAt:      time.Now(),
		},
		"empty title": {
			URL:           "https://example.com",
			Title:         " ",
			BookmarkCount: 1,
			PostedAt:      time.Now(),
		},
		"negative bookmark": {
			URL:           "https://example.com",
			Title:         "title",
			BookmarkCount: -10,
			PostedAt:      time.Now(),
		},
		"missing posted_at": {
			URL:           "https://example.com",
			Title:         "title",
			BookmarkCount: 1,
		},
		"invalid tag": {
			URL:           "https://example.com",
			Title:         "title",
			BookmarkCount: 1,
			PostedAt:      time.Now(),
			Tags: []Tagging{{
				TagID: uuid.Nil,
				Name:  "",
				Score: 2,
			}},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := New(params)
			require.ErrorIs(t, err, ErrInvalidEntry)
		})
	}
}

func TestListQueryNormalize(t *testing.T) {
	q := ListQuery{
		Tags:             []string{" Go ", "WEB"},
		MinBookmarkCount: 5,
		Limit:            500,
		Sort:             "",
	}

	require.NoError(t, q.Normalize())
	require.Equal(t, MaxLimit, q.Limit)
	require.Equal(t, SortNew, q.Sort)
	require.Equal(t, []string{"go", "web"}, q.Tags)
}

func TestListQueryNormalizeInvalid(t *testing.T) {
	tests := map[string]ListQuery{
		"negative offset": {Offset: -1},
		"negative bookmark filter": {
			MinBookmarkCount: -10,
		},
		"unsupported sort": {
			Sort: "unknown",
		},
	}

	for name, q := range tests {
		t.Run(name, func(t *testing.T) {
			err := q.Normalize()
			require.ErrorIs(t, err, ErrInvalidListQuery)
		})
	}
}
