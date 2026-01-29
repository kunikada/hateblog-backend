package entry

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	now := time.Now()
	validID := uuid.New()
	validTagID := uuid.New()

	tests := []struct {
		name    string
		params  Params
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid entry",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				Excerpt:       "Test excerpt",
				Subject:       "Test subject",
				BookmarkCount: 10,
				PostedAt:      now,
				Tags: []Tagging{
					{
						TagID: validTagID,
						Name:  "golang",
						Score: 80,
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "empty URL",
			params: Params{
				ID:            validID,
				URL:           "",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      now,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid URL",
			params: Params{
				ID:            validID,
				URL:           "not-a-valid-url",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      now,
			},
			wantErr: true,
			errMsg:  "invalid url",
		},
		{
			name: "empty title",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "",
				BookmarkCount: 10,
				PostedAt:      now,
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "whitespace-only title",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "   ",
				BookmarkCount: 10,
				PostedAt:      now,
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "negative bookmark count",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				BookmarkCount: -1,
				PostedAt:      now,
			},
			wantErr: true,
			errMsg:  "bookmark count must be >= 0",
		},
		{
			name: "zero posted_at",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      time.Time{},
			},
			wantErr: true,
			errMsg:  "posted_at is required",
		},
		{
			name: "invalid tag - nil TagID",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      now,
				Tags: []Tagging{
					{
						TagID: uuid.Nil,
						Name:  "golang",
						Score: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "tag id is required",
		},
		{
			name: "invalid tag - empty name",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      now,
				Tags: []Tagging{
					{
						TagID: validTagID,
						Name:  "",
						Score: 80,
					},
				},
			},
			wantErr: true,
			errMsg:  "tag name is required",
		},
		{
			name: "invalid tag - score too low",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      now,
				Tags: []Tagging{
					{
						TagID: validTagID,
						Name:  "golang",
						Score: -1,
					},
				},
			},
			wantErr: true,
			errMsg:  "tag score must be between 0 and 100",
		},
		{
			name: "invalid tag - score too high",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "Test Title",
				BookmarkCount: 10,
				PostedAt:      now,
				Tags: []Tagging{
					{
						TagID: validTagID,
						Name:  "golang",
						Score: 101,
					},
				},
			},
			wantErr: true,
			errMsg:  "tag score must be between 0 and 100",
		},
		{
			name: "trims whitespace from title",
			params: Params{
				ID:            validID,
				URL:           "https://example.com/article",
				Title:         "  Test Title  ",
				BookmarkCount: 10,
				PostedAt:      now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := New(tt.params)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, entry)
			} else {
				require.NoError(t, err)
				require.NotNil(t, entry)
				assert.Equal(t, tt.params.ID, entry.ID)
				assert.Equal(t, tt.params.URL, entry.URL)
				assert.Equal(t, tt.params.BookmarkCount, entry.BookmarkCount)
				assert.Equal(t, tt.params.PostedAt, entry.PostedAt)

				// Verify trimming
				if tt.name == "trims whitespace from title" {
					assert.Equal(t, "Test Title", entry.Title)
				}
			}
		})
	}
}

func TestEntry_Update(t *testing.T) {
	now := time.Now()
	validID := uuid.New()
	validTagID := uuid.New()

	// Create a valid initial entry
	initialParams := Params{
		ID:            validID,
		URL:           "https://example.com/initial",
		Title:         "Initial Title",
		Excerpt:       "Initial excerpt",
		Subject:       "Initial subject",
		BookmarkCount: 5,
		PostedAt:      now,
		Tags: []Tagging{
			{
				TagID: validTagID,
				Name:  "initial",
				Score: 50,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name       string
		params     Params
		wantErr    bool
		errMsg     string
		checkEntry func(t *testing.T, e *Entry)
	}{
		{
			name: "valid update",
			params: Params{
				URL:           "https://example.com/updated",
				Title:         "Updated Title",
				Excerpt:       "Updated excerpt",
				Subject:       "Updated subject",
				BookmarkCount: 20,
				PostedAt:      now.Add(time.Hour),
				Tags: []Tagging{
					{
						TagID: validTagID,
						Name:  "updated",
						Score: 90,
					},
				},
				UpdatedAt: now.Add(time.Hour),
			},
			wantErr: false,
			checkEntry: func(t *testing.T, e *Entry) {
				assert.Equal(t, "https://example.com/updated", e.URL)
				assert.Equal(t, "Updated Title", e.Title)
				assert.Equal(t, "Updated excerpt", e.Excerpt)
				assert.Equal(t, "Updated subject", e.Subject)
				assert.Equal(t, 20, e.BookmarkCount)
				assert.Equal(t, now.Add(time.Hour), e.PostedAt)
				require.Len(t, e.Tags, 1)
				assert.Equal(t, "updated", e.Tags[0].Name)
			},
		},
		{
			name: "empty URL",
			params: Params{
				URL:           "",
				Title:         "Updated Title",
				BookmarkCount: 10,
				PostedAt:      now,
				UpdatedAt:     now,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid URL",
			params: Params{
				URL:           "invalid-url",
				Title:         "Updated Title",
				BookmarkCount: 10,
				PostedAt:      now,
				UpdatedAt:     now,
			},
			wantErr: true,
			errMsg:  "invalid url",
		},
		{
			name: "empty title",
			params: Params{
				URL:           "https://example.com/updated",
				Title:         "",
				BookmarkCount: 10,
				PostedAt:      now,
				UpdatedAt:     now,
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "negative bookmark count",
			params: Params{
				URL:           "https://example.com/updated",
				Title:         "Updated Title",
				BookmarkCount: -1,
				PostedAt:      now,
				UpdatedAt:     now,
			},
			wantErr: true,
			errMsg:  "bookmark count must be >= 0",
		},
		{
			name: "zero posted_at",
			params: Params{
				URL:           "https://example.com/updated",
				Title:         "Updated Title",
				BookmarkCount: 10,
				PostedAt:      time.Time{},
				UpdatedAt:     now,
			},
			wantErr: true,
			errMsg:  "posted_at is required",
		},
		{
			name: "invalid tag in update",
			params: Params{
				URL:           "https://example.com/updated",
				Title:         "Updated Title",
				BookmarkCount: 10,
				PostedAt:      now,
				Tags: []Tagging{
					{
						TagID: uuid.Nil,
						Name:  "invalid",
						Score: 50,
					},
				},
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "tag id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh entry for each test
			entry, err := New(initialParams)
			require.NoError(t, err)

			err = entry.Update(tt.params)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				if tt.checkEntry != nil {
					tt.checkEntry(t, entry)
				}
			}
		})
	}
}

func TestNormalizeTaggings(t *testing.T) {
	tagID1 := uuid.New()
	tagID2 := uuid.New()

	tests := []struct {
		name  string
		input []Tagging
		want  []Tagging
	}{
		{
			name:  "empty tags",
			input: []Tagging{},
			want:  nil,
		},
		{
			name:  "nil tags",
			input: nil,
			want:  nil,
		},
		{
			name: "normalizes tag names",
			input: []Tagging{
				{
					TagID: tagID1,
					Name:  "  Golang  ",
					Score: 80,
				},
				{
					TagID: tagID2,
					Name:  "PROGRAMMING",
					Score: 60,
				},
			},
			want: []Tagging{
				{
					TagID: tagID1,
					Name:  "golang",
					Score: 80,
				},
				{
					TagID: tagID2,
					Name:  "programming",
					Score: 60,
				},
			},
		},
		{
			name: "preserves tag IDs and scores",
			input: []Tagging{
				{
					TagID: tagID1,
					Name:  "Test",
					Score: 95,
				},
			},
			want: []Tagging{
				{
					TagID: tagID1,
					Name:  "test",
					Score: 95,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTaggings(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListQuery_Normalize(t *testing.T) {
	tests := []struct {
		name    string
		query   ListQuery
		want    ListQuery
		wantErr bool
		errMsg  string
	}{
		{
			name:  "applies default limit",
			query: ListQuery{},
			want: ListQuery{
				Limit: DefaultLimit,
				Sort:  SortNew,
			},
			wantErr: false,
		},
		{
			name: "applies default sort",
			query: ListQuery{
				Limit: 50,
			},
			want: ListQuery{
				Limit: 50,
				Sort:  SortNew,
			},
			wantErr: false,
		},
		{
			name: "caps limit at MaxLimit",
			query: ListQuery{
				Limit: 2000,
			},
			want: ListQuery{
				Limit: MaxLimit,
				Sort:  SortNew,
			},
			wantErr: false,
		},
		{
			name: "respects MaxLimitOverride",
			query: ListQuery{
				Limit:            500,
				MaxLimitOverride: 1000,
			},
			want: ListQuery{
				Limit:            500,
				MaxLimitOverride: 1000,
				Sort:             SortNew,
			},
			wantErr: false,
		},
		{
			name: "caps at MaxLimitOverride",
			query: ListQuery{
				Limit:            2000,
				MaxLimitOverride: 1000,
			},
			want: ListQuery{
				Limit:            1000,
				MaxLimitOverride: 1000,
				Sort:             SortNew,
			},
			wantErr: false,
		},
		{
			name: "negative offset",
			query: ListQuery{
				Offset: -1,
			},
			wantErr: true,
			errMsg:  "offset must be >= 0",
		},
		{
			name: "negative min_bookmark_count",
			query: ListQuery{
				MinBookmarkCount: -1,
			},
			wantErr: true,
			errMsg:  "min_bookmark_count must be >= 0",
		},
		{
			name: "trims keyword",
			query: ListQuery{
				Keyword: "  test keyword  ",
			},
			want: ListQuery{
				Limit:   DefaultLimit,
				Sort:    SortNew,
				Keyword: "test keyword",
			},
			wantErr: false,
		},
		{
			name: "valid SortNew",
			query: ListQuery{
				Sort: SortNew,
			},
			want: ListQuery{
				Limit: DefaultLimit,
				Sort:  SortNew,
			},
			wantErr: false,
		},
		{
			name: "valid SortHot",
			query: ListQuery{
				Sort: SortHot,
			},
			want: ListQuery{
				Limit: DefaultLimit,
				Sort:  SortHot,
			},
			wantErr: false,
		},
		{
			name: "invalid sort type",
			query: ListQuery{
				Sort: "invalid",
			},
			wantErr: true,
			errMsg:  "unsupported sort",
		},
		{
			name: "converts posted_at_from to UTC",
			query: ListQuery{
				PostedAtFrom: time.Date(2024, 1, 1, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
			},
			want: ListQuery{
				Limit:        DefaultLimit,
				Sort:         SortNew,
				PostedAtFrom: time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "converts posted_at_to to UTC",
			query: ListQuery{
				PostedAtTo: time.Date(2024, 1, 1, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
			},
			want: ListQuery{
				Limit:      DefaultLimit,
				Sort:       SortNew,
				PostedAtTo: time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "posted_at_from must be before posted_at_to",
			query: ListQuery{
				PostedAtFrom: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				PostedAtTo:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
			errMsg:  "posted_at_from must be before posted_at_to",
		},
		{
			name: "posted_at_from equals posted_at_to",
			query: ListQuery{
				PostedAtFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				PostedAtTo:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
			errMsg:  "posted_at_from must be before posted_at_to",
		},
		{
			name: "normalizes tag names",
			query: ListQuery{
				Tags: []string{"  Golang  ", "PROGRAMMING", "web"},
			},
			want: ListQuery{
				Limit: DefaultLimit,
				Sort:  SortNew,
				Tags:  []string{"golang", "programming", "web"},
			},
			wantErr: false,
		},
		{
			name: "sorts tags",
			query: ListQuery{
				Tags: []string{"zebra", "alpha", "beta"},
			},
			want: ListQuery{
				Limit: DefaultLimit,
				Sort:  SortNew,
				Tags:  []string{"alpha", "beta", "zebra"},
			},
			wantErr: false,
		},
		{
			name: "single tag not sorted",
			query: ListQuery{
				Tags: []string{"golang"},
			},
			want: ListQuery{
				Limit: DefaultLimit,
				Sort:  SortNew,
				Tags:  []string{"golang"},
			},
			wantErr: false,
		},
		{
			name: "complex valid query",
			query: ListQuery{
				Tags:             []string{"programming", "golang"},
				MinBookmarkCount: 10,
				Offset:           20,
				Limit:            50,
				Sort:             SortHot,
				Keyword:          "tutorial",
				PostedAtFrom:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				PostedAtTo:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			want: ListQuery{
				Tags:             []string{"golang", "programming"},
				MinBookmarkCount: 10,
				Offset:           20,
				Limit:            50,
				Sort:             SortHot,
				Keyword:          "tutorial",
				PostedAtFrom:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				PostedAtTo:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.query
			err := q.Normalize()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.Limit, q.Limit)
				assert.Equal(t, tt.want.Sort, q.Sort)
				assert.Equal(t, tt.want.Keyword, q.Keyword)
				assert.Equal(t, tt.want.Tags, q.Tags)
				assert.Equal(t, tt.want.MinBookmarkCount, q.MinBookmarkCount)
				assert.Equal(t, tt.want.Offset, q.Offset)

				if !tt.want.PostedAtFrom.IsZero() {
					assert.True(t, q.PostedAtFrom.Equal(tt.want.PostedAtFrom))
				}
				if !tt.want.PostedAtTo.IsZero() {
					assert.True(t, q.PostedAtTo.Equal(tt.want.PostedAtTo))
				}
			}
		})
	}
}
