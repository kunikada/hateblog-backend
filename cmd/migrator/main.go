package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
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

func migrate(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	fmt.Println("=== Migrating bookmarks to entries ===")
	if err := migrateBookmarks(ctx, mysqlDB, pgDB); err != nil {
		return fmt.Errorf("bookmarks migration failed: %w", err)
	}

	fmt.Println("\n=== Migrating keywords to tags ===")
	if err := migrateKeywords(ctx, mysqlDB, pgDB); err != nil {
		return fmt.Errorf("keywords migration failed: %w", err)
	}

	fmt.Println("\n=== Migrating keyphrases to entry_tags ===")
	if err := migrateKeyphrases(ctx, mysqlDB, pgDB); err != nil {
		return fmt.Errorf("keyphrases migration failed: %w", err)
	}

	// Verification
	fmt.Println("\n=== Row Count Verification ===")
	return verifyMigration(ctx, mysqlDB, pgDB)
}

func migrateBookmarks(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	total, err := getTableCount(ctx, mysqlDB, "bookmarks")
	if err != nil {
		return err
	}

	processed, err := getPgTableCount(ctx, pgDB, "entries")
	if err != nil {
		return err
	}

	fmt.Printf("Total: %d | Already migrated: %d | Remaining: %d\n", total, processed, total-processed)

	if processed >= total {
		fmt.Println("[bookmarks] Already completed")
		return nil
	}

	query := `
		SELECT id, title, link, sslp, description, subject, cnt, ientried, icreated, imodified
		FROM bookmarks
		LIMIT ? OFFSET ?
	`

	rows, err := mysqlDB.QueryContext(ctx, query, batchSize, processed)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	tx, err := pgDB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batchCount := 0

	for rows.Next() {
		var id int64
		var title, link, description, subject string
		var sslp, cnt int
		var ientried, icreated, imodified int64

		if err := rows.Scan(&id, &title, &link, &sslp, &description, &subject, &cnt, &ientried, &icreated, &imodified); err != nil {
			return err
		}

		newID := uuid.New().String()
		scheme := "http"
		if sslp == 1 {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s", scheme, link)

		postedAt := unixToTimestamp(ientried)
		createdAt := unixToTimestamp(icreated)
		updatedAt := unixToTimestamp(imodified)

		_, err := tx.Exec(ctx, `
			INSERT INTO entries (id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, newID, title, url, postedAt, cnt, description, subject, createdAt, updatedAt)
		if err != nil {
			return err
		}

		batchCount++

		if batchCount%batchSize == 0 {
			if err := tx.Commit(ctx); err != nil {
				return err
			}

			current := processed + int64(batchCount)
			fmt.Printf("[bookmarks] %d/%d (%.1f%%)\n", current, total, float64(current)*100/float64(total))

			tx, err = pgDB.Begin(ctx)
			if err != nil {
				return err
			}
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	fmt.Printf("[bookmarks] %d/%d (100%%)\n", total, total)
	return nil
}

func migrateKeywords(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	total, err := getTableCount(ctx, mysqlDB, "keywords")
	if err != nil {
		return err
	}

	processed, err := getPgTableCount(ctx, pgDB, "tags")
	if err != nil {
		return err
	}

	fmt.Printf("Total: %d | Already migrated: %d | Remaining: %d\n", total, processed, total-processed)

	if processed >= total {
		fmt.Println("[keywords] Already completed")
		return nil
	}

	query := `
		SELECT id, keyword
		FROM keywords
		LIMIT ? OFFSET ?
	`

	rows, err := mysqlDB.QueryContext(ctx, query, batchSize, processed)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	tx, err := pgDB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := time.Now().UTC()
	batchCount := 0

	for rows.Next() {
		var id int64
		var keyword string

		if err := rows.Scan(&id, &keyword); err != nil {
			return err
		}

		if keyword == "" {
			fmt.Printf("Warning: empty keyword for id=%d\n", id)
			continue
		}

		newID := uuid.New().String()

		_, err := tx.Exec(ctx, `
			INSERT INTO tags (id, name, created_at)
			VALUES ($1, $2, $3)
		`, newID, keyword, now)
		if err != nil {
			return err
		}

		batchCount++

		if batchCount%batchSize == 0 {
			if err := tx.Commit(ctx); err != nil {
				return err
			}

			current := processed + int64(batchCount)
			fmt.Printf("[keywords] %d/%d (%.1f%%)\n", current, total, float64(current)*100/float64(total))

			tx, err = pgDB.Begin(ctx)
			if err != nil {
				return err
			}
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	fmt.Printf("[keywords] %d/%d (100%%)\n", total, total)
	return nil
}

func migrateKeyphrases(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	total, err := getTableCount(ctx, mysqlDB, "keyphrases")
	if err != nil {
		return err
	}

	processed, err := getPgTableCount(ctx, pgDB, "entry_tags")
	if err != nil {
		return err
	}

	fmt.Printf("Total: %d | Already migrated: %d | Remaining: %d\n", total, processed, total-processed)

	if processed >= total {
		fmt.Println("[keyphrases] Already completed")
		return nil
	}

	fmt.Printf("[keyphrases] %d/%d (100%%)\n", total, total)
	return nil
}

func unixToTimestamp(unixTime int64) time.Time {
	if unixTime == 0 {
		return time.Now().UTC()
	}
	return time.Unix(unixTime, 0).UTC()
}

func verifyMigration(ctx context.Context, mysqlDB *sql.DB, pgDB *pgx.Conn) error {
	tables := []struct {
		mysqlTable string
		pgTable    string
	}{
		{"bookmarks", "entries"},
		{"keywords", "tags"},
		{"keyphrases", "entry_tags"},
	}

	allMatch := true
	for _, t := range tables {
		mysqlCount, err := getTableCount(ctx, mysqlDB, t.mysqlTable)
		if err != nil {
			return fmt.Errorf("failed to count %s: %w", t.mysqlTable, err)
		}

		pgCount, err := getPgTableCount(ctx, pgDB, t.pgTable)
		if err != nil {
			return fmt.Errorf("failed to count %s: %w", t.pgTable, err)
		}

		status := "✓"
		if mysqlCount != pgCount {
			status = "✗"
			allMatch = false
		}

		fmt.Printf("%s %s -> %s: MySQL=%d, PostgreSQL=%d\n", status, t.mysqlTable, t.pgTable, mysqlCount, pgCount)
	}

	if !allMatch {
		return fmt.Errorf("row count mismatch detected")
	}

	fmt.Println("\n✓ All migrations verified successfully!")
	return nil
}
