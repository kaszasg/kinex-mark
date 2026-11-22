// Package store implements the SQLite persistence layer: connection,
// migrations and every query. Visibility and ownership are enforced in SQL.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/model"
)

//go:embed migrations/*.sql
var migrations embed.FS

// DBFileName is the database file inside the data directory.
const DBFileName = "app.db"

// Options configure a Store.
type Options struct {
	// Clock supplies every timestamp the store writes; nil means clock.Real.
	Clock clock.Clock
}

// Store holds the single database connection.
type Store struct {
	db    *sql.DB
	clock clock.Clock
}

// querier is the common subset of *sql.DB and *sql.Tx, so that helpers run
// both inside and outside a transaction (4.1).
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var (
	_ querier = (*sql.DB)(nil)
	_ querier = (*sql.Tx)(nil)
)

// Open opens or creates <dataDir>/app.db and applies pending migrations. If
// the migration fails, the database is closed and the error returned.
func Open(ctx context.Context, dataDir string, opts Options) (*Store, error) {
	path := filepath.Join(dataDir, DBFileName)
	query := url.Values{"_pragma": {
		"journal_mode(WAL)",
		"synchronous(NORMAL)",
		"busy_timeout(5000)",
		"foreign_keys(1)",
		// Deleted content is overwritten with zeros, so erased account data
		// (GDPR Art. 17, 6.11) does not linger in free space of app.db or in
		// its backups (deviation from TECHNOLOGY.md 5.1, plan 2.2).
		"secure_delete(1)",
	}}
	dsn := "file:" + path + "?" + query.Encode()
	// sql.Open only looks up the registered driver (which carries the
	// registered SQL functions); connections are opened through the
	// cancel-safe connector (cancelsafe.go).
	registered, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	drv := registered.Driver()
	_ = registered.Close()
	db := sql.OpenDB(cancelSafeConnector{drv: drv, dsn: dsn})
	// SQLite allows one writer at a time: a single connection avoids SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	clk := opts.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	return &Store{db: db, clock: clk}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Ping checks that the database is reachable.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// now returns the store clock's current time in UTC.
func (s *Store) now() time.Time { return s.clock.Now().UTC() }

// withTx runs fn in a transaction and commits it when fn returns nil. Inside
// fn only tx may be used: calling s.db would deadlock on the single
// connection (4.1).
func (s *Store) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// formatTime formats a timestamp for a TEXT column (model.TimeLayout).
func formatTime(t time.Time) string { return model.FormatTime(t) }

// formatNullTime formats a nullable timestamp for a TEXT column.
func formatNullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: model.FormatTime(*t), Valid: true}
}

// parseTime parses a TEXT timestamp column.
func parseTime(s string) (time.Time, error) { return model.ParseTime(s) }

// parseNullTime parses a nullable TEXT timestamp column.
func parseNullTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid {
		return nil, nil
	}
	t, err := model.ParseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// nullInt64 converts an optional id to a bind value (NULL when nil).
func nullInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// nullID binds an id that is NULL when not positive.
func nullID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

// int64Ptr converts a nullable integer column to a pointer.
func int64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

// stringPtr converts a nullable text column to a pointer.
func stringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

// notFound converts sql.ErrNoRows to model.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	return err
}

// migrate applies the embedded migrations that are not yet recorded, in
// name order, each in its own transaction (TECHNOLOGY.md 5.2).
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name       TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		var applied int
		err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration: %w", err)
		}
		if applied > 0 {
			continue
		}
		if err := applyMigration(ctx, db, name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string) error {
	body, err := migrations.ReadFile(name)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
		return err
	}
	return tx.Commit()
}
