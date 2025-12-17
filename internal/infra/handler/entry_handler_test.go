package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	appEntry "hateblog/internal/app/entry"
	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHandleNewEntries(t *testing.T) {
	repo := &mockEntryRepository{
		listResult: []*domainEntry.Entry{{
			ID:            uuid.New(),
			Title:         "Example",
			URL:           "https://example.com",
			BookmarkCount: 10,
			PostedAt:      time.Now(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}},
		countResult: 1,
	}

	service := appEntry.NewService(repo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewEntryHandler(service)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/entries/new?limit=10", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)

	var resp entryListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, int64(1), resp.Total)
	require.Equal(t, "Example", resp.Entries[0].Title)
}

func TestHandleNewEntries_InvalidParam(t *testing.T) {
	service := appEntry.NewService(&mockEntryRepository{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewEntryHandler(service)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/entries/new?limit=0", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, 400, rec.Code)
}

type mockEntryRepository struct {
	listResult  []*domainEntry.Entry
	countResult int64
}

func (m *mockEntryRepository) Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error) {
	return nil, nil
}

func (m *mockEntryRepository) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	return m.listResult, nil
}

func (m *mockEntryRepository) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	return m.countResult, nil
}

func (m *mockEntryRepository) Create(ctx context.Context, entry *domainEntry.Entry) error {
	return nil
}

func (m *mockEntryRepository) Update(ctx context.Context, entry *domainEntry.Entry) error {
	return nil
}

func (m *mockEntryRepository) Delete(ctx context.Context, id domainEntry.ID) error {
	return nil
}

func (m *mockEntryRepository) ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	return nil, nil
}
