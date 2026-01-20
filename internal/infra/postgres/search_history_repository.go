package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"hateblog/internal/domain/repository"
)

var _ repository.SearchHistoryRepository = (*SearchHistoryRepository)(nil)

// SearchHistoryRepository stores aggregated search queries.
type SearchHistoryRepository struct {
	pool *pgxpool.Pool
}

// NewSearchHistoryRepository creates a new repository.
func NewSearchHistoryRepository(pool *pgxpool.Pool) *SearchHistoryRepository {
	return &SearchHistoryRepository{pool: pool}
}

// Record increments the search count for the given query on the specified day.
func (r *SearchHistoryRepository) Record(ctx context.Context, query string, searchedAt time.Time) error {
	norm := strings.TrimSpace(query)
	if norm == "" {
		return fmt.Errorf("query is required")
	}
	date := time.Date(searchedAt.Year(), searchedAt.Month(), searchedAt.Day(), 0, 0, 0, 0, time.UTC)
	const stmt = `
INSERT INTO search_history (query, searched_at, count)
VALUES ($1, $2, 1)
ON CONFLICT (query, searched_at) DO UPDATE
SET count = search_history.count + 1`
	if _, err := r.pool.Exec(ctx, stmt, strings.ToLower(norm), date); err != nil {
		return fmt.Errorf("record search history: %w", err)
	}
	return nil
}
