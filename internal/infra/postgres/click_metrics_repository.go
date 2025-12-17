package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"
)

var _ repository.ClickMetricsRepository = (*ClickMetricsRepository)(nil)

// ClickMetricsRepository stores per-entry click counts.
type ClickMetricsRepository struct {
	pool *pgxpool.Pool
}

// NewClickMetricsRepository creates a new repository.
func NewClickMetricsRepository(pool *pgxpool.Pool) *ClickMetricsRepository {
	return &ClickMetricsRepository{pool: pool}
}

// Increment adds one click to the specified entry/date bucket.
func (r *ClickMetricsRepository) Increment(ctx context.Context, entryID entry.ID, clickedAt time.Time) error {
	if uuid.UUID(entryID) == uuid.Nil {
		return fmt.Errorf("entry id is required")
	}
	date := clickedAt.UTC().Truncate(24 * time.Hour)
	const stmt = `
INSERT INTO click_metrics (entry_id, clicked_at, count)
VALUES ($1, $2, 1)
ON CONFLICT (entry_id, clicked_at) DO UPDATE
SET count = click_metrics.count + 1`
	if _, err := r.pool.Exec(ctx, stmt, entryID, date); err != nil {
		return fmt.Errorf("increment click metrics: %w", err)
	}
	return nil
}
