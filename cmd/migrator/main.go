package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/caarlos0/env/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	domainEntry "hateblog/internal/domain/entry"
	"hateblog/internal/domain/tag"
)

// Config stores migration configuration values.
type Config struct {
	MySQLHost    string        `env:"MYSQL_HOST" envDefault:"localhost"`
	MySQLPort    int           `env:"MYSQL_PORT" envDefault:"3306"`
	MySQLUser    string        `env:"MYSQL_USER" envDefault:"root"`
	MySQLPass    string        `env:"MYSQL_PASSWORD" envDefault:""`
	MySQLDB      string        `env:"MYSQL_DB" envDefault:"hateblog_old"`
	MySQLTimeout time.Duration `env:"MYSQL_CONNECT_TIMEOUT" envDefault:"10s"`

	PostgresHost    string        `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort    int           `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser    string        `env:"POSTGRES_USER" envDefault:"hateblog"`
	PostgresPass    string        `env:"POSTGRES_PASSWORD" envDefault:"changeme"`
	PostgresDB      string        `env:"POSTGRES_DB" envDefault:"hateblog"`
	PostgresTimeout time.Duration `env:"POSTGRES_CONNECT_TIMEOUT" envDefault:"10s"`

	BatchSize int `env:"MIGRATION_BATCH_SIZE" envDefault:"1000"`
}

const batchSize = 1000

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	mysqlDB, err := connectMySQL(cfg)
	if err != nil {
		log.Fatalf("Failed to connect MySQL: %v", err)
	}
	defer func() {
		_ = mysqlDB.Close()
	}()

	pgDB, err := connectPostgres(cfg)
	if err != nil {
		log.Fatalf("Failed to connect PostgreSQL: %v", err)
	}

	ctx := context.Background()
	defer func() {
		_ = pgDB.Close(ctx)
	}()

	if err := migrate(ctx, mysqlDB, pgDB); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration completed successfully!")
}

func connectMySQL(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.MySQLUser, cfg.MySQLPass, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDB)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.MySQLTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func connectPostgres(cfg Config) (*pgx.Conn, error) {
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func getTableCount(ctx context.Context, db *sql.DB, table string) (int64, error) {
	var count int64
	row := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func getPgTableCount(ctx context.Context, db *pgx.Conn, table string) (int64, error) {
	var count int64
	row := db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func nullableText(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func getValidBookmarksCount(ctx context.Context, db *sql.DB) (int64, error) {
	var count int64
	row := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM bookmarks
		WHERE title IS NOT NULL AND title <> ''
		  AND link IS NOT NULL AND link <> ''
	`)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func migrate(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	fmt.Println("=== Migrating bookmarks, keywords, keyphrases ===")
	if err := migrateBatches(ctx, mysqlDB, pgDB); err != nil {
		return fmt.Errorf("batch migration failed: %w", err)
	}

	// Verification
	fmt.Println("\n=== Row Count Verification ===")
	return verifyMigration(ctx, mysqlDB, pgDB)
}

type bookmarkRow struct {
	id          int64
	title       sql.NullString
	link        sql.NullString
	sslp        int
	description sql.NullString
	subject     sql.NullString
	cnt         int
	ientried    int64
	icreated    int64
	imodified   int64
}

type keyphraseRow struct {
	bookmarkID int64
	keywordID  int64
	score      sql.NullInt64
}

type batchStats struct {
	processedBookmarks  int64
	insertedBookmarks   int64
	skippedBookmarks    int64
	insertedKeywords    int64
	insertedKeyphrases  int64
	skippedKeyphrases   int64
	skippedEmptyKeyword int64
}

func migrateBatches(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	total, err := getTableCount(ctx, mysqlDB, "bookmarks")
	if err != nil {
		return err
	}

	var (
		lastID          int64
		processed       int64
		totalSkipped    int64
		totalKeySkipped int64
	)

	lastID, err = getResumeLastID(ctx, mysqlDB, pgDB)
	if err != nil {
		return err
	}
	if lastID > 0 {
		fmt.Printf("[resume] Starting after bookmark id=%d based on latest entries.created_at\n", lastID)
	}

	for {
		bookmarks, err := fetchBookmarksBatch(ctx, mysqlDB, lastID, batchSize)
		if err != nil {
			return err
		}
		if len(bookmarks) == 0 {
			break
		}

		lastID = bookmarks[len(bookmarks)-1].id
		processed += int64(len(bookmarks))

		tx, err := pgDB.Begin(ctx)
		if err != nil {
			return err
		}

		stats, err := migrateBatch(ctx, mysqlDB, tx, bookmarks)
		if err != nil {
			rollbackTx(ctx, tx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		totalSkipped += stats.skippedBookmarks
		totalKeySkipped += stats.skippedKeyphrases

		progress := float64(0)
		if total > 0 {
			progress = float64(processed) * 100 / float64(total)
		}

		fmt.Printf("[batch] %d/%d (%.1f%%) | entries=%d | tags=%d | entry_tags=%d | skipped bookmarks=%d | skipped keyphrases=%d\n",
			processed, total, progress, stats.insertedBookmarks, stats.insertedKeywords, stats.insertedKeyphrases, stats.skippedBookmarks, stats.skippedKeyphrases)
	}

	if totalSkipped > 0 {
		fmt.Printf("[bookmarks] Warning: Skipped %d records due to NULL/empty required fields\n", totalSkipped)
	}
	if totalKeySkipped > 0 {
		fmt.Printf("[keyphrases] Warning: Skipped %d records due to missing mappings\n", totalKeySkipped)
	}

	return nil
}

func fetchBookmarksBatch(ctx context.Context, db *sql.DB, lastID int64, limit int) ([]bookmarkRow, error) {
	query := `
		SELECT id, title, link, sslp, description, subject, cnt, ientried, icreated, imodified
		FROM bookmarks
		WHERE id > ?
		ORDER BY id
		LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, lastID, limit)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var result []bookmarkRow
	for rows.Next() {
		var row bookmarkRow
		if err := rows.Scan(&row.id, &row.title, &row.link, &row.sslp, &row.description, &row.subject, &row.cnt, &row.ientried, &row.icreated, &row.imodified); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func migrateBatch(ctx context.Context, mysqlDB *sql.DB, tx pgx.Tx, bookmarks []bookmarkRow) (batchStats, error) {
	now := time.Now().UTC()

	stats := batchStats{
		processedBookmarks: int64(len(bookmarks)),
	}

	bookmarkToEntry := make(map[int64]string, len(bookmarks))
	validBookmarkIDs := make([]int64, 0, len(bookmarks))

	for _, bm := range bookmarks {
		if !bm.title.Valid || bm.title.String == "" {
			stats.skippedBookmarks++
			continue
		}
		if !bm.link.Valid || bm.link.String == "" {
			stats.skippedBookmarks++
			continue
		}

		newID := uuid.New().String()
		scheme := "http"
		if bm.sslp == 1 {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s", scheme, bm.link.String)

		postedAt := unixToTimestamp(bm.ientried)
		createdAt := unixToTimestamp(bm.icreated)
		updatedAt := unixToTimestamp(bm.imodified)

		var descriptionPtr *string
		if bm.description.Valid {
			descriptionPtr = &bm.description.String
		}

		var subjectPtr *string
		if bm.subject.Valid {
			subjectPtr = &bm.subject.String
		}

		description := ""
		if descriptionPtr != nil {
			description = *descriptionPtr
		}
		searchText := domainEntry.BuildSearchText(bm.title.String, description, url)

		_, err := tx.Exec(ctx, `
			INSERT INTO entries (id, title, url, posted_at, bookmark_count, excerpt, subject, search_text, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, newID, bm.title.String, url, postedAt, bm.cnt, descriptionPtr, subjectPtr, nullableText(searchText), createdAt, updatedAt)
		if err != nil {
			return stats, err
		}

		stats.insertedBookmarks++
		bookmarkToEntry[bm.id] = newID
		validBookmarkIDs = append(validBookmarkIDs, bm.id)
	}

	if len(validBookmarkIDs) == 0 {
		return stats, nil
	}

	keyphrases, err := fetchKeyphrasesByBookmarks(ctx, mysqlDB, validBookmarkIDs)
	if err != nil {
		return stats, err
	}
	if len(keyphrases) == 0 {
		return stats, nil
	}

	keywordIDs := make(map[int64]struct{})
	for _, kp := range keyphrases {
		keywordIDs[kp.keywordID] = struct{}{}
	}

	keywords, skippedEmpty, err := fetchKeywordsByIDs(ctx, mysqlDB, keywordIDs)
	if err != nil {
		return stats, err
	}
	stats.skippedEmptyKeyword += skippedEmpty

	keywordToTagID, insertedTags, err := ensureTags(ctx, tx, keywords, now)
	if err != nil {
		return stats, err
	}
	stats.insertedKeywords += insertedTags

	for _, kp := range keyphrases {
		entryID, entryExists := bookmarkToEntry[kp.bookmarkID]
		tagID, tagExists := keywordToTagID[kp.keywordID]
		if !entryExists || !tagExists {
			stats.skippedKeyphrases++
			continue
		}

		// Normalize score to 0-100 range (API should return 0-100, but clamp just in case)
		// NULL or values <= 0 become 0, values > 100 become 100
		score := 0
		if kp.score.Valid {
			rawScore := int(kp.score.Int64)
			if rawScore > 100 {
				score = 100
			} else if rawScore < 0 {
				score = 0
			} else {
				score = rawScore
			}
		}

		commandTag, err := tx.Exec(ctx, `
			INSERT INTO entry_tags (entry_id, tag_id, score, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (entry_id, tag_id) DO NOTHING
		`, entryID, tagID, score, now)
		if err != nil {
			return stats, err
		}
		stats.insertedKeyphrases += commandTag.RowsAffected()
	}

	return stats, nil
}

func fetchKeyphrasesByBookmarks(ctx context.Context, db *sql.DB, bookmarkIDs []int64) ([]keyphraseRow, error) {
	if len(bookmarkIDs) == 0 {
		return nil, nil
	}

	placeholders := buildPlaceholders(len(bookmarkIDs))
	args := make([]any, 0, len(bookmarkIDs))
	for _, id := range bookmarkIDs {
		args = append(args, id)
	}

	// #nosec G201 -- Placeholder list is built from IDs, not user input.
	query := fmt.Sprintf(`
		SELECT bookmark_id, keyword_id, score
		FROM keyphrases
		WHERE bookmark_id IN (%s)
	`, placeholders)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var result []keyphraseRow
	for rows.Next() {
		var row keyphraseRow
		if err := rows.Scan(&row.bookmarkID, &row.keywordID, &row.score); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchKeywordsByIDs(ctx context.Context, db *sql.DB, keywordIDs map[int64]struct{}) (map[int64]string, int64, error) {
	if len(keywordIDs) == 0 {
		return map[int64]string{}, 0, nil
	}

	ids := make([]int64, 0, len(keywordIDs))
	for id := range keywordIDs {
		ids = append(ids, id)
	}

	placeholders := buildPlaceholders(len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	// #nosec G201 -- Placeholder list is built from IDs, not user input.
	query := fmt.Sprintf(`
		SELECT id, keyword
		FROM keywords
		WHERE id IN (%s)
	`, placeholders)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer closeRows(rows)

	result := make(map[int64]string, len(ids))
	var skippedEmpty int64

	for rows.Next() {
		var id int64
		var keyword string
		if err := rows.Scan(&id, &keyword); err != nil {
			return nil, 0, err
		}
		if keyword == "" {
			skippedEmpty++
			continue
		}
		result[id] = keyword
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, skippedEmpty, nil
}

func ensureTags(ctx context.Context, tx pgx.Tx, keywords map[int64]string, now time.Time) (map[int64]string, int64, error) {
	if len(keywords) == 0 {
		return map[int64]string{}, 0, nil
	}

	nameSet := make(map[string]struct{}, len(keywords))
	names := make([]string, 0, len(keywords))
	for _, name := range keywords {
		// Sanitize and normalize the tag name
		sanitized := sanitizeUTF8(name)
		normalized := tag.NormalizeName(sanitized)

		// Skip empty tags after normalization
		if normalized == "" {
			continue
		}

		if _, exists := nameSet[normalized]; exists {
			continue
		}
		nameSet[normalized] = struct{}{}
		names = append(names, normalized)
	}

	var inserted int64
	for _, name := range names {
		newID := uuid.New().String()
		commandTag, err := tx.Exec(ctx, `
			INSERT INTO tags (id, name, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (name) DO NOTHING
		`, newID, name, now)
		if err != nil {
			return nil, 0, err
		}
		inserted += commandTag.RowsAffected()
	}

	rows, err := tx.Query(ctx, "SELECT id, name FROM tags WHERE name = ANY($1)", names)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	nameToID := make(map[string]string, len(names))
	for rows.Next() {
		var id string
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, 0, err
		}
		nameToID[name] = id
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	keywordToTag := make(map[int64]string, len(keywords))
	for keywordID, name := range keywords {
		// Apply the same normalization to match what was inserted
		sanitized := sanitizeUTF8(name)
		normalized := tag.NormalizeName(sanitized)

		if normalized == "" {
			continue
		}

		tagID, ok := nameToID[normalized]
		if !ok {
			return nil, 0, fmt.Errorf("tag id not found for keyword: %s", normalized)
		}
		keywordToTag[keywordID] = tagID
	}

	return keywordToTag, inserted, nil
}

// sanitizeUTF8 removes invalid UTF-8 sequences from a string
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// Convert to valid UTF-8 by removing invalid runes
	var builder strings.Builder
	builder.Grow(len(s))

	for _, r := range s {
		if r != utf8.RuneError {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func buildPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Printf("rows close failed: %v", err)
	}
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		log.Printf("rollback failed: %v", err)
	}
}

func getResumeLastID(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) (int64, error) {
	var latestCreatedAt time.Time
	if err := pgDB.QueryRow(ctx, `
		SELECT created_at
		FROM entries
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&latestCreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	rows, err := pgDB.Query(ctx, "SELECT url FROM entries WHERE created_at = $1", latestCreatedAt)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var httpLinks []string
	var httpsLinks []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return 0, err
		}
		if strings.HasPrefix(url, "https://") {
			httpsLinks = append(httpsLinks, strings.TrimPrefix(url, "https://"))
			continue
		}
		if strings.HasPrefix(url, "http://") {
			httpLinks = append(httpLinks, strings.TrimPrefix(url, "http://"))
			continue
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(httpLinks) == 0 && len(httpsLinks) == 0 {
		return 0, nil
	}

	var conditions []string
	args := make([]any, 0, len(httpLinks)+len(httpsLinks))
	if len(httpLinks) > 0 {
		conditions = append(conditions, fmt.Sprintf("(sslp = 0 AND link IN (%s))", buildPlaceholders(len(httpLinks))))
		for _, link := range httpLinks {
			args = append(args, link)
		}
	}
	if len(httpsLinks) > 0 {
		conditions = append(conditions, fmt.Sprintf("(sslp = 1 AND link IN (%s))", buildPlaceholders(len(httpsLinks))))
		for _, link := range httpsLinks {
			args = append(args, link)
		}
	}

	// #nosec G201 -- Placeholder list is built from URLs derived from stored entries.
	query := fmt.Sprintf(`
		SELECT MAX(id)
		FROM bookmarks
		WHERE title IS NOT NULL AND title <> ''
		  AND link IS NOT NULL AND link <> ''
		  AND (%s)
	`, strings.Join(conditions, " OR "))

	var maxID sql.NullInt64
	if err := mysqlDB.QueryRowContext(ctx, query, args...).Scan(&maxID); err != nil {
		return 0, err
	}
	if !maxID.Valid {
		return 0, nil
	}

	return maxID.Int64, nil
}

func unixToTimestamp(unixTime int64) time.Time {
	if unixTime == 0 {
		return time.Now().UTC()
	}
	return time.Unix(unixTime, 0).UTC()
}

func verifyMigration(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	mysqlCount, err := getValidBookmarksCount(ctx, mysqlDB)
	if err != nil {
		return fmt.Errorf("failed to count valid bookmarks: %w", err)
	}

	pgCount, err := getPgTableCount(ctx, pgDB, "entries")
	if err != nil {
		return fmt.Errorf("failed to count entries: %w", err)
	}

	status := "✓"
	if mysqlCount != pgCount {
		status = "✗"
	}

	fmt.Printf("%s bookmarks(valid) -> entries: MySQL=%d, PostgreSQL=%d\n", status, mysqlCount, pgCount)

	if mysqlCount != pgCount {
		return fmt.Errorf("row count mismatch detected")
	}

	fmt.Println("\n✓ Migration verified successfully!")
	return nil
}
