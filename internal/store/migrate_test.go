package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/model"
)

var migrationTables = []string{
	"users", "user_preferences", "sessions", "login_challenges", "recovery_codes",
	"login_throttle", "icons", "links", "tags", "link_tags", "link_shares",
	"link_user_state", "app_meta", "system_settings", "audit_log", "consent_records",
}

func openMigrationStore(t *testing.T, dir string) *Store {
	t.Helper()
	st, err := Open(context.Background(), dir, Options{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return st
}

func TestOpenMigratesEmptyDatabase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := openMigrationStore(t, dir)
	defer st.Close()
	ctx := context.Background()

	if err := st.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, DBFileName)); err != nil {
		t.Fatalf("database file: %v", err)
	}
	for _, table := range migrationTables {
		var n int
		err := st.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
		if err != nil || n != 1 {
			t.Errorf("table %s: count %d, err %v", table, n, err)
		}
	}
	var applied []string
	rows, err := st.db.QueryContext(ctx, `SELECT name FROM schema_migrations ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		applied = append(applied, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"migrations/0001_users.sql", "migrations/0002_auth.sql",
		"migrations/0003_links.sql", "migrations/0004_system.sql",
		"migrations/0005_fold_keys.sql",
	}
	if len(applied) != len(want) {
		t.Fatalf("applied = %v, want %v", applied, want)
	}
	for i := range want {
		if applied[i] != want[i] {
			t.Errorf("applied[%d] = %q, want %q", i, applied[i], want[i])
		}
	}

	var fk, journal string
	if err := st.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != "1" {
		t.Errorf("foreign_keys = %q, %v", fk, err)
	}
	if err := st.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journal); err != nil || journal != "wal" {
		t.Errorf("journal_mode = %q, %v", journal, err)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := openMigrationStore(t, dir)
	if _, err := st.db.Exec(`INSERT INTO app_meta (key, value) VALUES ('key_check', 'x')`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st = openMigrationStore(t, dir)
	defer st.Close()
	var n int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil || n != 5 {
		t.Fatalf("schema_migrations rows = %d, %v", n, err)
	}
	var v string
	if err := st.db.QueryRow(`SELECT value FROM app_meta WHERE key = 'key_check'`).Scan(&v); err != nil || v != "x" {
		t.Fatalf("data lost on reopen: %q, %v", v, err)
	}
}

// TestFoldKeysMigration checks 0005: a link stored with the old lower-cased
// keys gets the accent-free keys when the migration runs on reopen.
func TestFoldKeysMigration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := openMigrationStore(t, dir)
	now := formatTime(time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC))
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := st.db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO users (username, username_key, role, password_hash, password_changed_at)
		VALUES ('owner', 'owner', 'admin', 'h', ?)`, now)
	mustExec(`INSERT INTO links (owner_id, url, host, search_text, sort_label, created_at, updated_at)
		VALUES (1, 'https://example.com/', 'example.com', 'árvíz example.com', 'árvíz', ?, ?)`, now, now)
	mustExec(`DELETE FROM schema_migrations WHERE name = 'migrations/0005_fold_keys.sql'`)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st = openMigrationStore(t, dir)
	defer st.Close()
	var search, label string
	if err := st.db.QueryRow(`SELECT search_text, sort_label FROM links`).Scan(&search, &label); err != nil {
		t.Fatal(err)
	}
	if search != "arviz example.com" || label != "arviz" {
		t.Errorf("after 0005: search_text %q, sort_label %q", search, label)
	}
}

func TestOpenFailsOnUnusableDirectory(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "missing", "deeper")
	if _, err := Open(context.Background(), dir, Options{}); err == nil {
		t.Fatal("Open succeeded in a missing directory")
	}
}

func TestSchemaConstraints(t *testing.T) {
	t.Parallel()
	st := openMigrationStore(t, t.TempDir())
	defer st.Close()
	now := formatTime(time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC))

	exec := func(q string, args ...any) error {
		_, err := st.db.Exec(q, args...)
		return err
	}
	if err := exec(`INSERT INTO users (username, username_key, role, password_hash, password_changed_at)
		VALUES ('admin', 'admin', 'admin', 'h', ?)`, now); err != nil {
		t.Fatalf("valid user: %v", err)
	}
	bad := map[string]func() error{
		"short username": func() error {
			return exec(`INSERT INTO users (username, username_key, role, password_hash, password_changed_at)
				VALUES ('ab', 'ab', 'admin', 'h', ?)`, now)
		},
		"unknown role": func() error {
			return exec(`INSERT INTO users (username, username_key, role, password_hash, password_changed_at)
				VALUES ('root', 'root', 'root', 'h', ?)`, now)
		},
		"duplicate username key": func() error {
			return exec(`INSERT INTO users (username, username_key, role, password_hash, password_changed_at)
				VALUES ('ADMIN', 'admin', 'admin', 'h', ?)`, now)
		},
		"orphan link": func() error {
			return exec(`INSERT INTO links (owner_id, url, host, created_at, updated_at)
				VALUES (999, 'https://example.com/', 'example.com', ?, ?)`, now, now)
		},
		"bad tag color": func() error {
			return exec(`INSERT INTO tags (owner_id, name, name_key, color, created_at, updated_at)
				VALUES (1, 'x', 'x', 'red', ?, ?)`, now, now)
		},
		"invalid settings json": func() error {
			return exec(`INSERT INTO system_settings (key, value, updated_at) VALUES ('k', '{', ?)`, now)
		},
	}
	for name, fn := range bad {
		if err := fn(); err == nil {
			t.Errorf("%s: constraint not enforced", name)
		}
	}
}

func TestWithTxCommitsAndRollsBack(t *testing.T) {
	t.Parallel()
	st := openMigrationStore(t, t.TempDir())
	defer st.Close()
	ctx := context.Background()

	err := st.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO app_meta (key, value) VALUES ('a', '1')`)
		return err
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	boom := errors.New("boom")
	err = st.withTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO app_meta (key, value) VALUES ('b', '2')`); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("rollback error = %v", err)
	}
	var n int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_meta`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("app_meta rows = %d, %v", n, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := st.withTx(cancelled, func(*sql.Tx) error { return nil }); err == nil {
		t.Error("withTx on a cancelled context succeeded")
	}
}

func TestTimeHelpersAndClock(t *testing.T) {
	t.Parallel()
	fake := clock.NewFake(time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC))
	st, err := Open(context.Background(), t.TempDir(), Options{Clock: fake})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if !st.now().Equal(fake.Now()) {
		t.Errorf("now() = %v, want %v", st.now(), fake.Now())
	}

	ts := time.Date(2026, 2, 3, 4, 5, 6, 7_000_000, time.UTC)
	if got, err := parseTime(formatTime(ts)); err != nil || !got.Equal(ts) {
		t.Errorf("parseTime round trip: %v, %v", got, err)
	}
	if ns := formatNullTime(nil); ns.Valid {
		t.Error("formatNullTime(nil) is valid")
	}
	ns := formatNullTime(&ts)
	if !ns.Valid || ns.String != "2026-02-03T04:05:06.007Z" {
		t.Errorf("formatNullTime = %+v", ns)
	}
	if p, err := parseNullTime(sql.NullString{}); p != nil || err != nil {
		t.Errorf("parseNullTime(null) = %v, %v", p, err)
	}
	if p, err := parseNullTime(ns); err != nil || p == nil || !p.Equal(ts) {
		t.Errorf("parseNullTime = %v, %v", p, err)
	}
	if _, err := parseNullTime(sql.NullString{String: "x", Valid: true}); err == nil {
		t.Error("parseNullTime accepted garbage")
	}
	if !errors.Is(notFound(sql.ErrNoRows), model.ErrNotFound) {
		t.Error("notFound does not map sql.ErrNoRows")
	}
	other := errors.New("other")
	if notFound(other) != other {
		t.Error("notFound changed an unrelated error")
	}
}
