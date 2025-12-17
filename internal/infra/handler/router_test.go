package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"
	usecaseEntry "hateblog/internal/usecase/entry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRouter_EndToEndEntriesAndHealth(t *testing.T) {
	svc := usecaseEntry.NewService(&fakeRepo{
		list: []*domainEntry.Entry{{
			ID:            uuid.New(),
			Title:         "Sample",
			URL:           "https://example.com",
			BookmarkCount: 30,
			PostedAt:      time.Now(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}},
		count: 1,
	}, nil, nil)

	router := NewRouter(RouterConfig{
		EntryHandler: NewEntryHandler(svc),
		HealthHandler: &HealthHandler{
			DB:    &fakeHealthChecker{},
			Cache: &fakeHealthChecker{},
		},
	})

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/entries/new?limit=5")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var entryResp entryListResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &entryResp))
	require.Equal(t, int64(1), entryResp.Total)
	require.Len(t, entryResp.Entries, 1)

	healthResp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer healthResp.Body.Close()
	require.Equal(t, http.StatusOK, healthResp.StatusCode)
}

type fakeRepo struct {
	list  []*domainEntry.Entry
	count int64
}

func (f *fakeRepo) Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error) {
	return nil, nil
}

func (f *fakeRepo) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	return f.list, nil
}

func (f *fakeRepo) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	return f.count, nil
}

func (f *fakeRepo) Create(ctx context.Context, entry *domainEntry.Entry) error { return nil }
func (f *fakeRepo) Update(ctx context.Context, entry *domainEntry.Entry) error { return nil }
func (f *fakeRepo) Delete(ctx context.Context, id domainEntry.ID) error        { return nil }
func (f *fakeRepo) ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	return nil, nil
}

type fakeHealthChecker struct{}

func (f *fakeHealthChecker) HealthCheck(ctx context.Context) error {
	return nil
}
