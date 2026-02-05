package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	usecaseFavicon "hateblog/internal/usecase/favicon"

	"github.com/go-chi/chi/v5"
)

func TestFaviconHandlerSuccess(t *testing.T) {
	handler := NewFaviconHandler(newTestService([]byte{1, 2}, "image/png", nil))
	r := chi.NewRouter()
	r.Route(testAPIBasePath, func(r chi.Router) {
		handler.RegisterRoutes(r)
	})

	req := httptest.NewRequest(http.MethodGet, apiPath("/favicons?domain=example.com"), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected content type %s", rec.Header().Get("Content-Type"))
	}
	if body := rec.Body.Bytes(); len(body) != 2 || body[0] != 1 {
		t.Fatalf("unexpected body %v", body)
	}
}

func TestFaviconHandlerBadRequest(t *testing.T) {
	handler := NewFaviconHandler(newTestService(nil, "", nil))
	r := chi.NewRouter()
	r.Route(testAPIBasePath, func(r chi.Router) {
		handler.RegisterRoutes(r)
	})

	req := httptest.NewRequest(http.MethodGet, apiPath("/favicons?domain="+url.QueryEscape("bad host")), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestFaviconHandlerTooEarly(t *testing.T) {
	handler := NewFaviconHandler(newTestServiceWithLimiter([]byte{1}, "image/png", nil, &testLimiter{allow: false}))
	r := chi.NewRouter()
	r.Route(testAPIBasePath, func(r chi.Router) {
		handler.RegisterRoutes(r)
	})

	req := httptest.NewRequest(http.MethodGet, apiPath("/favicons?domain=example.com"), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooEarly {
		t.Fatalf("expected 425, got %d", rec.Code)
	}
}

func newTestService(data []byte, ctype string, fetchErr error) *usecaseFavicon.Service {
	fetcher := &testFetcher{
		data:  data,
		ctype: ctype,
		err:   fetchErr,
	}
	cache := &testCache{}
	return usecaseFavicon.NewService(fetcher, cache, nil, nil)
}

func newTestServiceWithLimiter(data []byte, ctype string, fetchErr error, limiter usecaseFavicon.Limiter) *usecaseFavicon.Service {
	fetcher := &testFetcher{
		data:  data,
		ctype: ctype,
		err:   fetchErr,
	}
	cache := &testCache{}
	return usecaseFavicon.NewService(fetcher, cache, limiter, nil)
}

type testFetcher struct {
	data  []byte
	ctype string
	err   error
}

func (t *testFetcher) Fetch(ctx context.Context, domain string) ([]byte, string, error) {
	if t.err != nil {
		return nil, "", t.err
	}
	return t.data, t.ctype, nil
}

type testCache struct {
	data  []byte
	ctype string
}

func (t *testCache) BuildKey(domain string) (string, error) {
	return "favicon:" + domain, nil
}

func (t *testCache) Get(ctx context.Context, key string) ([]byte, string, bool, error) {
	if len(t.data) == 0 {
		return nil, "", false, nil
	}
	return t.data, t.ctype, true, nil
}

func (t *testCache) Set(ctx context.Context, key string, data []byte, contentType string) error {
	t.data = data
	t.ctype = contentType
	return nil
}

func (t *testCache) SetNegative(ctx context.Context, key string) error {
	return nil
}

func (t *testCache) IsNegative(ctx context.Context, key string) (bool, error) {
	return false, nil
}

type testLimiter struct {
	allow bool
	err   error
}

func (t *testLimiter) Allow(ctx context.Context, domain string) (bool, error) {
	return t.allow, t.err
}
