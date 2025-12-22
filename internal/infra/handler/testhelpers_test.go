package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"
	domainTag "hateblog/internal/domain/tag"
	usecaseEntry "hateblog/internal/usecase/entry"
	usecaseSearch "hateblog/internal/usecase/search"
	usecaseTag "hateblog/internal/usecase/tag"
)

// testServer wraps httptest.Server for integration testing.
type testServer struct {
	*httptest.Server
	router http.Handler
}

// newTestServer creates a test HTTP server with the given handlers.
func newTestServer(cfg RouterConfig) *testServer {
	router := NewRouter(cfg)
	srv := httptest.NewServer(router)
	return &testServer{
		Server: srv,
		router: router,
	}
}

// get performs a GET request to the test server.
func (ts *testServer) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}

// decodeJSON decodes response body as JSON.
func decodeJSON(t *testing.T, resp *http.Response, dest interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
}

// assertStatus checks HTTP status code.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}
}

// assertContentType checks Content-Type header.
func assertContentType(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	got := resp.Header.Get("Content-Type")
	if got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

// Mock repository implementations for isolated handler testing

// mockEntryRepository is a mock implementation of entry repository.
type mockEntryRepository struct {
	listFunc  func(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error)
	countFunc func(ctx context.Context, query domainEntry.ListQuery) (int64, error)
	entries   []*domainEntry.Entry
	total     int64
}

func (m *mockEntryRepository) Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error) {
	return nil, nil
}

func (m *mockEntryRepository) List(ctx context.Context, query domainEntry.ListQuery) ([]*domainEntry.Entry, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, query)
	}
	return m.entries, nil
}

func (m *mockEntryRepository) Count(ctx context.Context, query domainEntry.ListQuery) (int64, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, query)
	}
	return m.total, nil
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

// mockTagRepository is a mock implementation of tag repository.
type mockTagRepository struct {
	getByNameFunc           func(ctx context.Context, name string) (*domainTag.Tag, error)
	listFunc                func(ctx context.Context, limit, offset int) ([]domainTag.Tag, error)
	incrementViewHistoryFunc func(ctx context.Context, tagID domainTag.ID, viewedAt time.Time) error
}

func (m *mockTagRepository) GetByName(ctx context.Context, name string) (*domainTag.Tag, error) {
	if m.getByNameFunc != nil {
		return m.getByNameFunc(ctx, name)
	}
	return nil, fmt.Errorf("tag not found")
}

func (m *mockTagRepository) List(ctx context.Context, limit, offset int) ([]domainTag.Tag, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockTagRepository) IncrementViewHistory(ctx context.Context, tagID domainTag.ID, viewedAt time.Time) error {
	if m.incrementViewHistoryFunc != nil {
		return m.incrementViewHistoryFunc(ctx, tagID, viewedAt)
	}
	return nil
}

// mockSearchHistoryRepository is a mock implementation of search history repository.
type mockSearchHistoryRepository struct {
	recordFunc func(ctx context.Context, query string, searchedAt time.Time) error
}

func (m *mockSearchHistoryRepository) Record(ctx context.Context, query string, searchedAt time.Time) error {
	if m.recordFunc != nil {
		return m.recordFunc(ctx, query, searchedAt)
	}
	return nil
}

// mockHealthChecker is a mock implementation of health checker.
type mockHealthChecker struct {
	healthCheckFunc func(ctx context.Context) error
}

func (m *mockHealthChecker) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

// Service builder helpers

// newTestEntryService creates an entry service with mock repository.
func newTestEntryService(repo *mockEntryRepository) *usecaseEntry.Service {
	return usecaseEntry.NewService(repo, nil, nil, nil)
}

// newTestTagService creates a tag service with mock repository.
func newTestTagService(repo *mockTagRepository) *usecaseTag.Service {
	return usecaseTag.NewService(repo, nil)
}

// newTestSearchService creates a search service with mock repositories.
func newTestSearchService(entryRepo *mockEntryRepository, historyRepo *mockSearchHistoryRepository) *usecaseSearch.Service {
	return usecaseSearch.NewService(entryRepo, historyRepo, nil, nil)
}

// Test data factories

// newTestEntry creates a test entry with sensible defaults.
func newTestEntry(id domainEntry.ID, title string, bookmarkCount int) *domainEntry.Entry {
	now := time.Now().UTC()
	return &domainEntry.Entry{
		ID:            id,
		Title:         title,
		URL:           fmt.Sprintf("https://example.com/%s", id.String()),
		BookmarkCount: bookmarkCount,
		PostedAt:      now.Add(-1 * time.Hour),
		Excerpt:       "This is an excerpt",
		Subject:       "technology",
		Tags: []domainEntry.Tagging{
			{
				TagID: uuid.New(),
				Name:  "tech",
				Score: 0.8,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// newTestEntryWithTags creates a test entry with custom tags.
func newTestEntryWithTags(id domainEntry.ID, title string, bookmarkCount int, tags []domainEntry.Tagging) *domainEntry.Entry {
	entry := newTestEntry(id, title, bookmarkCount)
	entry.Tags = tags
	return entry
}

// newTestTag creates a test tag.
func newTestTag(id domainTag.ID, name string) *domainTag.Tag {
	return &domainTag.Tag{
		ID:   id,
		Name: name,
	}
}

// newTestTagging creates a test tagging relationship.
func newTestTagging(tagID domainTag.ID, name string, score float64) domainEntry.Tagging {
	return domainEntry.Tagging{
		TagID: tagID,
		Name:  name,
		Score: score,
	}
}

// buildTestListResult creates a test list result.
func buildTestListResult(entries []*domainEntry.Entry, total int64) usecaseEntry.ListResult {
	return usecaseEntry.ListResult{
		Entries: entries,
		Total:   total,
	}
}

// buildTestSearchResult creates a test search result.
func buildTestSearchResult(query string, entries []*domainEntry.Entry, total int64, limit, offset int) usecaseSearch.Result {
	return usecaseSearch.Result{
		Query:   query,
		Entries: entries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}
}

// Assertion helpers

// assertEntryListResponse validates the structure of entry list response.
func assertEntryListResponse(t *testing.T, resp *http.Response) entryListResponse {
	t.Helper()
	assertStatus(t, resp, http.StatusOK)
	assertContentType(t, resp, "application/json")

	var result entryListResponse
	decodeJSON(t, resp, &result)
	return result
}

// assertErrorResponse validates error response structure.
func assertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int) map[string]string {
	t.Helper()
	assertStatus(t, resp, expectedStatus)
	assertContentType(t, resp, "application/json")

	var result map[string]string
	decodeJSON(t, resp, &result)
	if _, ok := result["error"]; !ok {
		t.Error("error response missing 'error' field")
	}
	return result
}

// assertEntryResponse validates a single entry response.
func assertEntryResponse(t *testing.T, got entryResponse, want *domainEntry.Entry) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
	if got.Title != want.Title {
		t.Errorf("Title = %q, want %q", got.Title, want.Title)
	}
	if got.URL != want.URL {
		t.Errorf("URL = %q, want %q", got.URL, want.URL)
	}
	if got.BookmarkCount != want.BookmarkCount {
		t.Errorf("BookmarkCount = %d, want %d", got.BookmarkCount, want.BookmarkCount)
	}
	if len(got.Tags) != len(want.Tags) {
		t.Errorf("Tags count = %d, want %d", len(got.Tags), len(want.Tags))
	}
}

// assertPagination validates pagination fields.
func assertPagination(t *testing.T, total int64, limit, offset int, wantTotal int64, wantLimit, wantOffset int) {
	t.Helper()
	if total != wantTotal {
		t.Errorf("Total = %d, want %d", total, wantTotal)
	}
	if limit != wantLimit {
		t.Errorf("Limit = %d, want %d", limit, wantLimit)
	}
	if offset != wantOffset {
		t.Errorf("Offset = %d, want %d", offset, wantOffset)
	}
}
