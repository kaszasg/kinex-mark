package store

import (
	"context"
	"database/sql/driver"
	"errors"
)

// The modernc.org/sqlite driver interrupts a running statement when the
// query context ends. Up to v1.39.0 (stmt.query in sqlite.go), when that
// interrupt raced with a query that had already produced its rows, the
// driver dropped the rows without finalizing the prepared statement.
// sqlite3_close_v2 then kept the connection alive as a zombie, and the
// database file stayed open after Store.Close (on Windows app.db could no
// longer be removed).
//
// Re-checked against v1.62.0 (stmt.query in stmt.go): the deferred block now
// closes the rows before it replaces them with the context error (upstream
// fix in v1.40.1), so that leak no longer exists. A probe running the
// workload of TestCancelledQueriesReleaseDatabaseFile on a plain
// sql.Open pool kept app.db open in 0 of 2000 runs with v1.62.0, against
// 23-30 of 200 runs with v1.39.0. The wrapper stays as defence in depth,
// and its regression test stays as the guard for later driver versions.
//
// cancelSafeConn hands the driver a context without cancellation for every
// row-returning query, so the driver never starts that interrupt. The
// original context still governs everything database/sql does around the
// call: it stops waiting for the connection, and it closes the rows (which
// finalizes the statement) once the context ends. Statements without rows
// keep the original context; their interrupt path finalizes correctly.
//
// The wrapper forwards every optional interface the driver documents on
// sqlite.NewConnector for connection wrappers, and Unwrap exposes the
// driver connection (sqlite.Limit looks through a wrapper that way).

// errConnLacksContext is returned when the driver connection does not offer
// the context-aware interfaces the store relies on.
var errConnLacksContext = errors.New("sqlite connection does not support contexts")

// sqliteConn is the set of driver interfaces a modernc connection implements
// and database/sql uses.
type sqliteConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.ExecerContext
	driver.QueryerContext
	driver.Pinger
	driver.SessionResetter
	driver.Validator
}

// sqliteStmt is the set of driver interfaces a modernc statement implements.
type sqliteStmt interface {
	driver.Stmt
	driver.StmtExecContext
	driver.StmtQueryContext
}

// cancelSafeConnector opens driver connections wrapped in cancelSafeConn.
type cancelSafeConnector struct {
	drv driver.Driver
	dsn string
}

// Connect opens a connection and wraps it.
func (c cancelSafeConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := c.drv.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	full, ok := conn.(sqliteConn)
	if !ok {
		_ = conn.Close()
		return nil, errConnLacksContext
	}
	return cancelSafeConn{full}, nil
}

// Driver returns the underlying driver.
func (c cancelSafeConnector) Driver() driver.Driver { return c.drv }

// cancelSafeConn forwards every call to the driver connection; only queries
// run with a context that cannot be cancelled.
type cancelSafeConn struct {
	sqliteConn
}

// Unwrap returns the driver connection.
func (c cancelSafeConn) Unwrap() driver.Conn { return c.sqliteConn }

// QueryContext runs the query without handing cancellation to the driver.
func (c cancelSafeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.sqliteConn.QueryContext(context.WithoutCancel(ctx), query, args)
}

// PrepareContext prepares a statement whose queries are cancel-safe as well.
func (c cancelSafeConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	st, err := c.sqliteConn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	full, ok := st.(sqliteStmt)
	if !ok {
		return st, nil
	}
	return cancelSafeStmt{full}, nil
}

// cancelSafeStmt forwards every call to the driver statement; only queries
// run with a context that cannot be cancelled.
type cancelSafeStmt struct {
	sqliteStmt
}

// QueryContext runs the statement without handing cancellation to the driver.
func (s cancelSafeStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.sqliteStmt.QueryContext(context.WithoutCancel(ctx), args)
}
