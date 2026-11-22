package store

import (
	"context"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCancelledQueriesReleaseDatabaseFile is the regression test for the
// leaked file handle: queries cancelled at varying points must not keep
// app.db open after Close. With modernc.org/sqlite v1.39.0 and without the
// cancel-safe connector about one run in eight kept the file open for good
// (Windows refuses the remove); v1.62.0 no longer leaks (cancelsafe.go).
func TestCancelledQueriesReleaseDatabaseFile(t *testing.T) {
	const runs = 60
	for i := 0; i < runs; i++ {
		dir := t.TempDir()
		st, err := Open(context.Background(), dir, Options{})
		if err != nil {
			t.Fatalf("run %d: Open: %v", i, err)
		}
		runCtx, cancel := context.WithCancel(context.Background())
		timer := time.AfterFunc(time.Duration(50+i*20)*time.Microsecond, cancel)
		for j := 0; j < 50 && runCtx.Err() == nil; j++ {
			_, _ = st.Settings(runCtx)
			_, _ = st.DueFetches(runCtx, 10)
			_, _ = st.ArchiveInactive(runCtx, 90)
			_, _ = st.Purge(runCtx, 0)
		}
		timer.Stop()
		cancel()
		if err := st.Close(); err != nil {
			t.Fatalf("run %d: Close: %v", i, err)
		}
		if err := os.Remove(filepath.Join(dir, DBFileName)); err != nil {
			t.Fatalf("run %d: database file still open after Close: %v", i, err)
		}
	}
}

// TestCancelledQueryFailsWithContextError checks that a query with an ended
// context still fails, although the driver never sees the cancellation.
func TestCancelledQueryFailsWithContextError(t *testing.T) {
	st, _ := newTestStore(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := st.Settings(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Settings with a cancelled context: err = %v, want context.Canceled", err)
	}
	if _, err := st.Settings(context.Background()); err != nil {
		t.Fatalf("Settings after a cancelled query: %v", err)
	}
}

type ctxKey struct{}

// fakeSQLiteConn records the contexts the wrapper passes on.
type fakeSQLiteConn struct {
	queryCtx   context.Context
	execCtx    context.Context
	prepareErr error
	stmt       driver.Stmt
	closed     bool
}

func (c *fakeSQLiteConn) Prepare(string) (driver.Stmt, error) { return c.stmt, nil }
func (c *fakeSQLiteConn) Close() error                        { c.closed = true; return nil }
func (c *fakeSQLiteConn) Begin() (driver.Tx, error)           { return nil, errors.New("not used") }
func (c *fakeSQLiteConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return nil, errors.New("not used")
}

func (c *fakeSQLiteConn) PrepareContext(context.Context, string) (driver.Stmt, error) {
	if c.prepareErr != nil {
		return nil, c.prepareErr
	}
	return c.stmt, nil
}

func (c *fakeSQLiteConn) ExecContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	c.execCtx = ctx
	return driver.RowsAffected(1), nil
}

func (c *fakeSQLiteConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	c.queryCtx = ctx
	return nil, nil
}

func (c *fakeSQLiteConn) Ping(context.Context) error         { return nil }
func (c *fakeSQLiteConn) ResetSession(context.Context) error { return nil }
func (c *fakeSQLiteConn) IsValid() bool                      { return true }

// fakeSQLiteStmt records the contexts the wrapper passes on.
type fakeSQLiteStmt struct {
	queryCtx context.Context
	execCtx  context.Context
}

func (s *fakeSQLiteStmt) Close() error                               { return nil }
func (s *fakeSQLiteStmt) NumInput() int                              { return -1 }
func (s *fakeSQLiteStmt) Exec([]driver.Value) (driver.Result, error) { return nil, nil }
func (s *fakeSQLiteStmt) Query([]driver.Value) (driver.Rows, error)  { return nil, nil }
func (s *fakeSQLiteStmt) ExecContext(ctx context.Context, _ []driver.NamedValue) (driver.Result, error) {
	s.execCtx = ctx
	return driver.RowsAffected(1), nil
}

func (s *fakeSQLiteStmt) QueryContext(ctx context.Context, _ []driver.NamedValue) (driver.Rows, error) {
	s.queryCtx = ctx
	return nil, nil
}

// plainStmtOnly implements only driver.Stmt.
type plainStmtOnly struct{}

func (plainStmtOnly) Close() error                               { return nil }
func (plainStmtOnly) NumInput() int                              { return -1 }
func (plainStmtOnly) Exec([]driver.Value) (driver.Result, error) { return nil, nil }
func (plainStmtOnly) Query([]driver.Value) (driver.Rows, error)  { return nil, nil }

// plainConn implements only driver.Conn.
type plainConn struct{ closed bool }

func (c *plainConn) Prepare(string) (driver.Stmt, error) { return plainStmtOnly{}, nil }
func (c *plainConn) Close() error                        { c.closed = true; return nil }
func (c *plainConn) Begin() (driver.Tx, error)           { return nil, errors.New("not used") }

// fakeDriver returns a fixed connection or error.
type fakeDriver struct {
	conn driver.Conn
	err  error
	dsn  string
}

func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	d.dsn = name
	if d.err != nil {
		return nil, d.err
	}
	return d.conn, nil
}

func TestCancelSafeConnQueriesWithoutCancellation(t *testing.T) {
	inner := &fakeSQLiteConn{stmt: &fakeSQLiteStmt{}}
	drv := &fakeDriver{conn: inner}
	connector := cancelSafeConnector{drv: drv, dsn: "file:x.db"}
	if connector.Driver() != drv {
		t.Fatal("Driver() does not return the wrapped driver")
	}
	conn, err := connector.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if drv.dsn != "file:x.db" {
		t.Fatalf("driver opened %q, want file:x.db", drv.dsn)
	}
	c := conn.(cancelSafeConn)
	if c.Unwrap() != driver.Conn(inner) {
		t.Fatal("Unwrap() does not return the driver connection")
	}

	live, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{}, "v"))
	defer cancel()
	if _, err := c.QueryContext(live, "SELECT 1", nil); err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	if inner.queryCtx.Done() != nil {
		t.Error("the driver query got a cancellable context")
	}
	if inner.queryCtx.Value(ctxKey{}) != "v" {
		t.Error("the driver query lost the context values")
	}
	if _, err := c.ExecContext(live, "DELETE FROM x", nil); err != nil {
		t.Fatalf("ExecContext: %v", err)
	}
	if inner.execCtx.Done() == nil {
		t.Error("ExecContext should pass the original context on")
	}

	ended, endCancel := context.WithCancel(context.Background())
	endCancel()
	inner.queryCtx = nil
	if _, err := c.QueryContext(ended, "SELECT 1", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("QueryContext with an ended context: err = %v, want context.Canceled", err)
	}
	if inner.queryCtx != nil {
		t.Error("the driver was called with an ended context")
	}
}

func TestCancelSafeStmtQueriesWithoutCancellation(t *testing.T) {
	innerStmt := &fakeSQLiteStmt{}
	c := cancelSafeConn{&fakeSQLiteConn{stmt: innerStmt}}
	st, err := c.PrepareContext(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("PrepareContext: %v", err)
	}
	s, ok := st.(cancelSafeStmt)
	if !ok {
		t.Fatalf("PrepareContext returned %T, want cancelSafeStmt", st)
	}
	live, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := s.QueryContext(live, nil); err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	if innerStmt.queryCtx.Done() != nil {
		t.Error("the driver statement got a cancellable context")
	}
	if _, err := s.ExecContext(live, nil); err != nil {
		t.Fatalf("ExecContext: %v", err)
	}
	if innerStmt.execCtx.Done() == nil {
		t.Error("ExecContext should pass the original context on")
	}
	ended, endCancel := context.WithCancel(context.Background())
	endCancel()
	innerStmt.queryCtx = nil
	if _, err := s.QueryContext(ended, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("QueryContext with an ended context: err = %v, want context.Canceled", err)
	}
	if innerStmt.queryCtx != nil {
		t.Error("the driver statement was called with an ended context")
	}
}

func TestCancelSafeConnectorErrors(t *testing.T) {
	openErr := errors.New("open failed")
	if _, err := (cancelSafeConnector{drv: &fakeDriver{err: openErr}}).Connect(context.Background()); !errors.Is(err, openErr) {
		t.Fatalf("Connect with a failing driver: err = %v", err)
	}

	plain := &plainConn{}
	if _, err := (cancelSafeConnector{drv: &fakeDriver{conn: plain}}).Connect(context.Background()); !errors.Is(err, errConnLacksContext) {
		t.Fatalf("Connect with a plain connection: err = %v, want errConnLacksContext", err)
	}
	if !plain.closed {
		t.Error("the unusable connection was not closed")
	}

	prepErr := errors.New("prepare failed")
	c := cancelSafeConn{&fakeSQLiteConn{prepareErr: prepErr}}
	if _, err := c.PrepareContext(context.Background(), "SELECT 1"); !errors.Is(err, prepErr) {
		t.Fatalf("PrepareContext error: err = %v", err)
	}

	c = cancelSafeConn{&fakeSQLiteConn{stmt: plainStmtOnly{}}}
	st, err := c.PrepareContext(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("PrepareContext: %v", err)
	}
	if _, ok := st.(plainStmtOnly); !ok {
		t.Fatalf("a statement without context support should be returned as is, got %T", st)
	}
}
