package postgres

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainArchive "hateblog/internal/domain/archive"
	"hateblog/internal/domain/entry"
	"hateblog/internal/domain/repository"
)

var _ repository.EntryRepository = (*EntryRepository)(nil)

// EntryRepository implements repository.EntryRepository backed by PostgreSQL.
type EntryRepository struct {
	pool *pgxpool.Pool
}

// NewEntryRepository creates a new EntryRepository.
func NewEntryRepository(pool *pgxpool.Pool) *EntryRepository {
	return &EntryRepository{pool: pool}
}

// Create inserts a new entry.
func (r *EntryRepository) Create(ctx context.Context, e *entry.Entry) error {
	if e == nil {
		return fmt.Errorf("entry is nil")
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}
	searchText := entry.BuildSearchText(e.Title, e.Excerpt, e.URL)
	const query = `
INSERT INTO entries (id, title, url, posted_at, bookmark_count, excerpt, subject, search_text, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		e.ID,
		e.Title,
		e.URL,
		e.PostedAt,
		e.BookmarkCount,
		nullableString(e.Excerpt),
		nullableString(e.Subject),
		nullableString(searchText),
		e.CreatedAt,
		e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert entry: %w", err)
	}
	return nil
}

// Update updates an existing entry.
func (r *EntryRepository) Update(ctx context.Context, e *entry.Entry) error {
	if e == nil {
		return fmt.Errorf("entry is nil")
	}
	if e.ID == uuid.Nil {
		return fmt.Errorf("entry id is required")
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = time.Now()
	}
	searchText := entry.BuildSearchText(e.Title, e.Excerpt, e.URL)
	const query = `
UPDATE entries
SET title = $1,
	url = $2,
	posted_at = $3,
	bookmark_count = $4,
	excerpt = $5,
	subject = $6,
	search_text = $7,
	updated_at = $8
WHERE id = $9`

	_, err := r.pool.Exec(ctx, query,
		e.Title,
		e.URL,
		e.PostedAt,
		e.BookmarkCount,
		nullableString(e.Excerpt),
		nullableString(e.Subject),
		nullableString(searchText),
		e.UpdatedAt,
		e.ID,
	)
	if err != nil {
		return fmt.Errorf("update entry: %w", err)
	}
	return nil
}

// Delete removes an entry by ID.
func (r *EntryRepository) Delete(ctx context.Context, id entry.ID) error {
	if id == uuid.Nil {
		return fmt.Errorf("entry id is required")
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	return nil
}

// Get retrieves a single entry by ID.
func (r *EntryRepository) Get(ctx context.Context, id entry.ID) (*entry.Entry, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("entry id is required")
	}
	const query = `
SELECT id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at
FROM entries
WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	ent, err := scanEntry(row)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, fmt.Errorf("entry not found: %w", err)
		}
		return nil, err
	}
	if err := r.loadTags(ctx, []*entry.Entry{ent}); err != nil {
		return nil, err
	}
	return ent, nil
}

// List returns entries that match the query.
func (r *EntryRepository) List(ctx context.Context, q entry.ListQuery) ([]*entry.Entry, error) {
	query := q
	if err := query.Normalize(); err != nil {
		return nil, err
	}
	sql, args := buildListEntriesSQL(query, false)
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	entries, err := scanEntries(rows)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return entries, nil
	}

	if err := r.loadTags(ctx, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Count returns the number of entries matching the query.
func (r *EntryRepository) Count(ctx context.Context, q entry.ListQuery) (int64, error) {
	query := q
	if err := query.Normalize(); err != nil {
		return 0, err
	}
	sql, args := buildListEntriesSQL(query, true)
	var count int64
	if err := r.pool.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count entries: %w", err)
	}
	return count, nil
}

// ListAndCount returns entries and the total count. When the page has no rows, it falls back to Count.
func (r *EntryRepository) ListAndCount(ctx context.Context, q entry.ListQuery) ([]*entry.Entry, int64, error) {
	query := q
	if err := query.Normalize(); err != nil {
		return nil, 0, err
	}
	sql, args := buildListEntriesWithTotalSQL(query)
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list entries with total: %w", err)
	}
	defer rows.Close()

	var (
		entries []*entry.Entry
		total   int64
	)
	for rows.Next() {
		ent := &entry.Entry{}
		var excerpt, subject *string
		var rowTotal int64
		if err := rows.Scan(
			&ent.ID,
			&ent.Title,
			&ent.URL,
			&ent.PostedAt,
			&ent.BookmarkCount,
			&excerpt,
			&subject,
			&ent.CreatedAt,
			&ent.UpdatedAt,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("scan entry with total: %w", err)
		}
		if excerpt != nil {
			ent.Excerpt = *excerpt
		}
		if subject != nil {
			ent.Subject = *subject
		}
		total = rowTotal
		entries = append(entries, ent)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(entries) == 0 {
		count, err := r.Count(ctx, query)
		if err != nil {
			return nil, 0, err
		}
		return entries, count, nil
	}

	if err := r.loadTags(ctx, entries); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// ListArchiveCounts aggregates entries per day ordered by date desc.
func (r *EntryRepository) ListArchiveCounts(ctx context.Context, minBookmarkCount int) ([]repository.ArchiveCount, error) {
	if err := domainArchive.ValidateMinUsers(minBookmarkCount); err != nil {
		return nil, err
	}
	const query = `
SELECT day, count
FROM archive_counts
WHERE threshold = $1
ORDER BY day DESC`

	rows, err := r.pool.Query(ctx, query, minBookmarkCount)
	if err != nil {
		return nil, fmt.Errorf("archive counts: %w", err)
	}
	defer rows.Close()

	var items []repository.ArchiveCount
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("scan archive count: %w", err)
		}
		items = append(items, repository.ArchiveCount{
			Date:  day,
			Count: count,
		})
	}
	return items, rows.Err()
}

func (r *EntryRepository) loadTags(ctx context.Context, entries []*entry.Entry) error {
	ids := make([]uuid.UUID, 0, len(entries))
	entryByID := make(map[uuid.UUID]*entry.Entry, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
		entryByID[e.ID] = e
	}

	const query = `
SELECT et.entry_id, t.id, t.name, et.score
FROM entry_tags et
INNER JOIN tags t ON t.id = et.tag_id
WHERE et.entry_id = ANY($1)`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entryID uuid.UUID
		var tagID uuid.UUID
		var name string
		var score int
		if err := rows.Scan(&entryID, &tagID, &name, &score); err != nil {
			return fmt.Errorf("scan entry tags: %w", err)
		}

		ent := entryByID[entryID]
		ent.Tags = append(ent.Tags, entry.Tagging{
			TagID: tagID,
			Name:  name,
			Score: score,
		})
	}

	return rows.Err()
}

func buildListEntriesSQL(q entry.ListQuery, countOnly bool) (string, []any) {
	if q.Keyword != "" {
		return buildKeywordSearchSQL(q, countOnly, false)
	}
	var columns string
	if countOnly {
		columns = "COUNT(1)"
	} else {
		columns = "id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at"
	}

	builder := strings.Builder{}
	builder.WriteString("SELECT ")
	builder.WriteString(columns)
	builder.WriteString(" FROM entries e")

	var conditions []string
	var args []any
	argPos := 1

	if q.MinBookmarkCount > 0 {
		conditions = append(conditions, fmt.Sprintf("bookmark_count >= $%d", argPos))
		args = append(args, q.MinBookmarkCount)
		argPos++
	}

	if len(q.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM entry_tags et
			INNER JOIN tags t ON t.id = et.tag_id
			WHERE et.entry_id = e.id AND t.name = ANY($%d)
		)`, argPos))
		args = append(args, q.Tags)
		argPos++
	}

	if !q.PostedAtFrom.IsZero() {
		conditions = append(conditions, fmt.Sprintf("posted_at >= $%d", argPos))
		args = append(args, q.PostedAtFrom)
		argPos++
	}

	if !q.PostedAtTo.IsZero() {
		conditions = append(conditions, fmt.Sprintf("posted_at < $%d", argPos))
		args = append(args, q.PostedAtTo)
		argPos++
	}

	if len(conditions) > 0 {
		builder.WriteString(" WHERE ")
		builder.WriteString(strings.Join(conditions, " AND "))
	}

	if countOnly {
		return builder.String(), args
	}

	switch q.Sort {
	case entry.SortHot:
		builder.WriteString(" ORDER BY bookmark_count DESC, posted_at DESC")
	default:
		builder.WriteString(" ORDER BY posted_at DESC")
	}

	builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, q.Limit, q.Offset)

	return builder.String(), args
}

func buildListEntriesWithTotalSQL(q entry.ListQuery) (string, []any) {
	if q.Keyword != "" {
		return buildKeywordSearchSQL(q, false, true)
	}
	columns := "id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at, COUNT(1) OVER() AS total"

	builder := strings.Builder{}
	builder.WriteString("SELECT ")
	builder.WriteString(columns)
	builder.WriteString(" FROM entries e")

	var conditions []string
	var args []any
	argPos := 1

	if q.MinBookmarkCount > 0 {
		conditions = append(conditions, fmt.Sprintf("bookmark_count >= $%d", argPos))
		args = append(args, q.MinBookmarkCount)
		argPos++
	}

	if len(q.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM entry_tags et
			INNER JOIN tags t ON t.id = et.tag_id
			WHERE et.entry_id = e.id AND t.name = ANY($%d)
		)`, argPos))
		args = append(args, q.Tags)
		argPos++
	}

	if !q.PostedAtFrom.IsZero() {
		conditions = append(conditions, fmt.Sprintf("posted_at >= $%d", argPos))
		args = append(args, q.PostedAtFrom)
		argPos++
	}

	if !q.PostedAtTo.IsZero() {
		conditions = append(conditions, fmt.Sprintf("posted_at < $%d", argPos))
		args = append(args, q.PostedAtTo)
		argPos++
	}

	if len(conditions) > 0 {
		builder.WriteString(" WHERE ")
		builder.WriteString(strings.Join(conditions, " AND "))
	}

	switch q.Sort {
	case entry.SortHot:
		builder.WriteString(" ORDER BY bookmark_count DESC, posted_at DESC")
	default:
		builder.WriteString(" ORDER BY posted_at DESC")
	}

	builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, q.Limit, q.Offset)

	return builder.String(), args
}

func scanEntries(rows pgx.Rows) ([]*entry.Entry, error) {
	var entries []*entry.Entry
	for rows.Next() {
		ent, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ent)
	}
	return entries, rows.Err()
}

func scanEntry(row pgx.Row) (*entry.Entry, error) {
	ent := &entry.Entry{}
	var excerpt, subject *string
	if err := row.Scan(
		&ent.ID,
		&ent.Title,
		&ent.URL,
		&ent.PostedAt,
		&ent.BookmarkCount,
		&excerpt,
		&subject,
		&ent.CreatedAt,
		&ent.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan entry: %w", err)
	}
	if excerpt != nil {
		ent.Excerpt = *excerpt
	}
	if subject != nil {
		ent.Subject = *subject
	}
	return ent, nil
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func buildKeywordSearchSQL(q entry.ListQuery, countOnly bool, withTotal bool) (string, []any) {
	const candidateLimit = 2000

	terms := splitSearchTerms(q.Keyword)
	if len(terms) == 0 {
		clone := q
		clone.Keyword = ""
		if withTotal {
			return buildListEntriesWithTotalSQL(clone)
		}
		return buildListEntriesSQL(clone, countOnly)
	}

	termsAny := make([]string, 0, len(terms))
	enWords := make([]string, 0, len(terms))
	for _, term := range terms {
		if term == "" {
			continue
		}
		normalized := strings.ToLower(term)
		termsAny = append(termsAny, normalized)
		if isASCIIWord(normalized) {
			enWords = append(enWords, normalized)
		}
	}

	enRegex := "a^"
	if len(enWords) > 0 {
		enRegex = englishWordRegex(enWords)
	}

	var columns string
	switch {
	case countOnly:
		columns = "COUNT(1)"
	case withTotal:
		columns = "id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at, COUNT(1) OVER() AS total"
	default:
		columns = "id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at"
	}

	builder := strings.Builder{}
	args := make([]any, 0, 10)
	argPos := 1

	builder.WriteString("WITH params AS (SELECT ")
	builder.WriteString(fmt.Sprintf("$%d::text[] AS terms_any, ", argPos))
	args = append(args, termsAny)
	argPos++
	builder.WriteString(fmt.Sprintf("$%d::text[] AS en_words, ", argPos))
	args = append(args, enWords)
	argPos++
	builder.WriteString(fmt.Sprintf("$%d::text AS en_regex)", argPos))
	args = append(args, enRegex)
	argPos++

	builder.WriteString(" , candidates AS (SELECT e.* FROM entries e, params p WHERE ")
	builder.WriteString(" (SELECT bool_and(e.search_text LIKE '%' || t || '%') FROM unnest(p.terms_any) t)")

	if q.MinBookmarkCount > 0 {
		builder.WriteString(fmt.Sprintf(" AND e.bookmark_count >= $%d", argPos))
		args = append(args, q.MinBookmarkCount)
		argPos++
	}

	if len(q.Tags) > 0 {
		builder.WriteString(fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM entry_tags et
			INNER JOIN tags t ON t.id = et.tag_id
			WHERE et.entry_id = e.id AND t.name = ANY($%d)
		)`, argPos))
		args = append(args, q.Tags)
		argPos++
	}

	if !q.PostedAtFrom.IsZero() {
		builder.WriteString(fmt.Sprintf(" AND e.posted_at >= $%d", argPos))
		args = append(args, q.PostedAtFrom)
		argPos++
	}

	if !q.PostedAtTo.IsZero() {
		builder.WriteString(fmt.Sprintf(" AND e.posted_at < $%d", argPos))
		args = append(args, q.PostedAtTo)
		argPos++
	}

	builder.WriteString(fmt.Sprintf(" LIMIT %d)", candidateLimit))

	builder.WriteString(" SELECT ")
	builder.WriteString(columns)
	builder.WriteString(" FROM candidates c, params p WHERE ")
	builder.WriteString(" (cardinality(p.en_words) = 0 OR (SELECT COUNT(DISTINCT m[1]) FROM regexp_matches(c.search_text, p.en_regex, 'g') m) = cardinality(p.en_words))")

	if countOnly {
		return builder.String(), args
	}

	switch q.Sort {
	case entry.SortHot:
		builder.WriteString(" ORDER BY c.bookmark_count DESC, c.posted_at DESC")
	default:
		builder.WriteString(" ORDER BY c.posted_at DESC")
	}

	builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, q.Limit, q.Offset)

	return builder.String(), args
}

func splitSearchTerms(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		return unicode.IsSpace(r)
	})
}

func isASCIIWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func englishWordRegex(words []string) string {
	if len(words) == 0 {
		return "a^"
	}
	escaped := make([]string, 0, len(words))
	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}
		escaped = append(escaped, regexp.QuoteMeta(trimmed))
	}
	if len(escaped) == 0 {
		return "a^"
	}
	return fmt.Sprintf("\\m(%s)\\M", strings.Join(escaped, "|"))
}
