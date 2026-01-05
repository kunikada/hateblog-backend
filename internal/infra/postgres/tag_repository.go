package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"hateblog/internal/domain/repository"
	"hateblog/internal/domain/tag"
)

var _ repository.TagRepository = (*TagRepository)(nil)

// TagRepository implements repository.TagRepository backed by PostgreSQL.
type TagRepository struct {
	pool *pgxpool.Pool
}

// NewTagRepository creates a new TagRepository.
func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

// Get retrieves a tag by ID.
func (r *TagRepository) Get(ctx context.Context, id tag.ID) (*tag.Tag, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("tag id is required")
	}
	const query = `SELECT id, name FROM tags WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	var result tag.Tag
	if err := row.Scan(&result.ID, &result.Name); err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &result, nil
}

// GetByName retrieves a tag by normalized name.
func (r *TagRepository) GetByName(ctx context.Context, name string) (*tag.Tag, error) {
	norm := tag.NormalizeName(name)
	if norm == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	const query = `SELECT id, name FROM tags WHERE name = $1`
	row := r.pool.QueryRow(ctx, query, norm)
	var result tag.Tag
	if err := row.Scan(&result.ID, &result.Name); err != nil {
		return nil, fmt.Errorf("get tag by name: %w", err)
	}
	return &result, nil
}

// List returns tags ordered by name.
func (r *TagRepository) List(ctx context.Context, limit, offset int) ([]tag.Tag, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM tags ORDER BY name ASC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []tag.Tag
	for rows.Next() {
		var t tag.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// Upsert inserts or updates a tag by name.
func (r *TagRepository) Upsert(ctx context.Context, t *tag.Tag) error {
	if t == nil {
		return fmt.Errorf("tag is nil")
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Name == "" {
		return fmt.Errorf("tag name is required")
	}

	const query = `
INSERT INTO tags (id, name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET
	name = EXCLUDED.name
RETURNING id, name`

	now := time.Now().UTC()
	if err := r.pool.QueryRow(ctx, query, t.ID, tag.NormalizeName(t.Name), now).Scan(&t.ID, &t.Name); err != nil {
		return fmt.Errorf("upsert tag: %w", err)
	}
	return nil
}

// Delete removes a tag.
func (r *TagRepository) Delete(ctx context.Context, id tag.ID) error {
	if id == uuid.Nil {
		return fmt.Errorf("tag id is required")
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}

// IncrementViewHistory adds one view count for the specified tag/date.
func (r *TagRepository) IncrementViewHistory(ctx context.Context, tagID tag.ID, viewedAt time.Time) error {
	if tagID == uuid.Nil {
		return fmt.Errorf("tag id is required")
	}
	date := viewedAt.UTC().Truncate(24 * time.Hour)
	const query = `
INSERT INTO tag_view_history (tag_id, viewed_at, count)
VALUES ($1, $2, 1)
ON CONFLICT (tag_id, viewed_at) DO UPDATE
SET count = tag_view_history.count + 1`
	if _, err := r.pool.Exec(ctx, query, tagID, date); err != nil {
		return fmt.Errorf("increment tag view history: %w", err)
	}
	return nil
}

// GetTrending returns tags from recent popular entries, ordered by occurrence count.
func (r *TagRepository) GetTrending(ctx context.Context, hours int, minBookmarkCount int, limit int) ([]tag.TrendingTag, error) {
	if hours <= 0 {
		hours = 24
	}
	if limit <= 0 {
		limit = 20
	}

	const query = `
SELECT
    t.id,
    t.name,
    COUNT(DISTINCT e.id) AS occurrence_count,
    (SELECT COUNT(DISTINCT et2.entry_id) FROM entry_tags et2 WHERE et2.tag_id = t.id) AS entry_count
FROM tags t
INNER JOIN entry_tags et ON et.tag_id = t.id
INNER JOIN entries e ON e.id = et.entry_id
WHERE e.posted_at >= NOW() - $1::interval
  AND e.bookmark_count >= $2
GROUP BY t.id, t.name
ORDER BY occurrence_count DESC, t.name ASC
LIMIT $3`

	interval := fmt.Sprintf("%d hours", hours)
	rows, err := r.pool.Query(ctx, query, interval, minBookmarkCount, limit)
	if err != nil {
		return nil, fmt.Errorf("get trending tags: %w", err)
	}
	defer rows.Close()

	var tags []tag.TrendingTag
	for rows.Next() {
		var t tag.TrendingTag
		if err := rows.Scan(&t.ID, &t.Name, &t.OccurrenceCount, &t.EntryCount); err != nil {
			return nil, fmt.Errorf("scan trending tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// GetClicked returns tags from recently clicked entries, ordered by click count.
func (r *TagRepository) GetClicked(ctx context.Context, days int, limit int) ([]tag.ClickedTag, error) {
	if days <= 0 {
		days = 7
	}
	if limit <= 0 {
		limit = 20
	}

	const query = `
SELECT
    t.id,
    t.name,
    SUM(cm.count) AS click_count,
    (SELECT COUNT(DISTINCT et2.entry_id) FROM entry_tags et2 WHERE et2.tag_id = t.id) AS entry_count
FROM tags t
INNER JOIN entry_tags et ON et.tag_id = t.id
INNER JOIN click_metrics cm ON cm.entry_id = et.entry_id
WHERE cm.clicked_at >= CURRENT_DATE - $1::interval
GROUP BY t.id, t.name
ORDER BY click_count DESC, t.name ASC
LIMIT $2`

	interval := fmt.Sprintf("%d days", days)
	rows, err := r.pool.Query(ctx, query, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("get clicked tags: %w", err)
	}
	defer rows.Close()

	var tags []tag.ClickedTag
	for rows.Next() {
		var t tag.ClickedTag
		if err := rows.Scan(&t.ID, &t.Name, &t.ClickCount, &t.EntryCount); err != nil {
			return nil, fmt.Errorf("scan clicked tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
