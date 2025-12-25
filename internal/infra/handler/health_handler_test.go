package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestHealthHandler_Healthy(t *testing.T) {
	mockDB := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockCache := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}

	handler := &HealthHandler{
		DB:    mockDB,
		Cache: mockCache,
	}

	ts := newTestServer(RouterConfig{
		HealthHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/health"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusOK)
	assertContentType(t, resp, "application/json")

	var result map[string]interface{}
	decodeJSON(t, resp, &result)

	if result["status"] != "healthy" {
		t.Errorf("status = %v, want %q", result["status"], "healthy")
	}

	components, ok := result["components"].([]interface{})
	if !ok {
		t.Fatal("components should be an array")
	}

	if len(components) != 2 {
		t.Errorf("got %d components, want 2", len(components))
	}

	// Verify all components are healthy
	for _, comp := range components {
		c := comp.(map[string]interface{})
		if c["status"] != "healthy" {
			t.Errorf("component %s status = %v, want %q", c["name"], c["status"], "healthy")
		}
		if _, hasError := c["error"]; hasError {
			t.Errorf("healthy component %s should not have error field", c["name"])
		}
	}
}

func TestHealthHandler_UnhealthyDB(t *testing.T) {
	mockDB := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return fmt.Errorf("connection refused")
		},
	}
	mockCache := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}

	handler := &HealthHandler{
		DB:    mockDB,
		Cache: mockCache,
	}

	ts := newTestServer(RouterConfig{
		HealthHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/health"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusServiceUnavailable)
	assertContentType(t, resp, "application/json")

	var result map[string]interface{}
	decodeJSON(t, resp, &result)

	if result["status"] != "unhealthy" {
		t.Errorf("status = %v, want %q", result["status"], "unhealthy")
	}

	components, ok := result["components"].([]interface{})
	if !ok {
		t.Fatal("components should be an array")
	}

	// Find database component and verify it's unhealthy
	var foundUnhealthyDB bool
	for _, comp := range components {
		c := comp.(map[string]interface{})
		if c["name"] == "database" {
			if c["status"] != "unhealthy" {
				t.Errorf("database status = %v, want %q", c["status"], "unhealthy")
			}
			if _, hasError := c["error"]; !hasError {
				t.Error("unhealthy database should have error field")
			}
			foundUnhealthyDB = true
		}
	}

	if !foundUnhealthyDB {
		t.Error("database component not found in response")
	}
}

func TestHealthHandler_UnhealthyCache(t *testing.T) {
	mockDB := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return nil
		},
	}
	mockCache := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return fmt.Errorf("redis connection timeout")
		},
	}

	handler := &HealthHandler{
		DB:    mockDB,
		Cache: mockCache,
	}

	ts := newTestServer(RouterConfig{
		HealthHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/health"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusServiceUnavailable)
	assertContentType(t, resp, "application/json")

	var result map[string]interface{}
	decodeJSON(t, resp, &result)

	if result["status"] != "unhealthy" {
		t.Errorf("status = %v, want %q", result["status"], "unhealthy")
	}

	components, ok := result["components"].([]interface{})
	if !ok {
		t.Fatal("components should be an array")
	}

	// Find redis component and verify it's unhealthy
	var foundUnhealthyCache bool
	for _, comp := range components {
		c := comp.(map[string]interface{})
		if c["name"] == "redis" {
			if c["status"] != "unhealthy" {
				t.Errorf("redis status = %v, want %q", c["status"], "unhealthy")
			}
			if _, hasError := c["error"]; !hasError {
				t.Error("unhealthy redis should have error field")
			}
			foundUnhealthyCache = true
		}
	}

	if !foundUnhealthyCache {
		t.Error("redis component not found in response")
	}
}

func TestHealthHandler_AllUnhealthy(t *testing.T) {
	mockDB := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return fmt.Errorf("database down")
		},
	}
	mockCache := &mockHealthChecker{
		healthCheckFunc: func(ctx context.Context) error {
			return fmt.Errorf("cache down")
		},
	}

	handler := &HealthHandler{
		DB:    mockDB,
		Cache: mockCache,
	}

	ts := newTestServer(RouterConfig{
		HealthHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/health"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusServiceUnavailable)

	var result map[string]interface{}
	decodeJSON(t, resp, &result)

	if result["status"] != "unhealthy" {
		t.Errorf("status = %v, want %q", result["status"], "unhealthy")
	}

	components, ok := result["components"].([]interface{})
	if !ok {
		t.Fatal("components should be an array")
	}

	// Verify all components are unhealthy
	unhealthyCount := 0
	for _, comp := range components {
		c := comp.(map[string]interface{})
		if c["status"] == "unhealthy" {
			unhealthyCount++
			if _, hasError := c["error"]; !hasError {
				t.Errorf("unhealthy component %s should have error field", c["name"])
			}
		}
	}

	if unhealthyCount != 2 {
		t.Errorf("got %d unhealthy components, want 2", unhealthyCount)
	}
}

func TestHealthHandler_NilDependencies(t *testing.T) {
	handler := &HealthHandler{
		DB:    nil,
		Cache: nil,
	}

	ts := newTestServer(RouterConfig{
		HealthHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/health"))
	defer resp.Body.Close()

	assertStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	decodeJSON(t, resp, &result)

	if result["status"] != "healthy" {
		t.Errorf("status = %v, want %q", result["status"], "healthy")
	}

	components, ok := result["components"].([]interface{})
	if !ok {
		t.Fatal("components should be an array")
	}

	if len(components) != 0 {
		t.Errorf("got %d components, want 0 (nil dependencies should not be checked)", len(components))
	}
}

func TestHealthHandler_ResponseFields(t *testing.T) {
	handler := &HealthHandler{
		DB:    &mockHealthChecker{},
		Cache: &mockHealthChecker{},
	}

	ts := newTestServer(RouterConfig{
		HealthHandler: handler,
	})
	defer ts.Close()

	resp := ts.get(t, apiPath("/health"))
	defer resp.Body.Close()

	var result map[string]interface{}
	decodeJSON(t, resp, &result)

	// Verify required fields exist
	requiredFields := []string{"status", "components", "checked_at"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("response missing required field: %s", field)
		}
	}

	// Verify checked_at is a valid timestamp
	if _, ok := result["checked_at"].(string); !ok {
		t.Error("checked_at should be a string (ISO 8601 timestamp)")
	}
}
