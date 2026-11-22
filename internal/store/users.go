package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// userColumns selects a model.User (scanUser) from the users table "u".
const userColumns = `u.id, u.username, u.username_key, u.display_name, u.role,
	u.must_change_password, u.is_active, u.totp_secret IS NOT NULL,
	u.password_changed_at, u.last_login_at, u.created_at, u.updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

// userScan holds the raw columns of userColumns.
type userScan struct {
	u                                       model.User
	role                                    string
	passwordChangedAt, createdAt, updatedAt string
	lastLoginAt                             sql.NullString
}

func (us *userScan) dest() []any {
	return []any{&us.u.ID, &us.u.Username, &us.u.UsernameKey, &us.u.DisplayName, &us.role,
		&us.u.MustChangePassword, &us.u.Active, &us.u.TOTPEnabled,
		&us.passwordChangedAt, &us.lastLoginAt, &us.createdAt, &us.updatedAt}
}

func (us *userScan) user() (model.User, error) {
	u := us.u
	u.Role = model.Role(us.role)
	var err error
	if u.PasswordChangedAt, err = parseTime(us.passwordChangedAt); err != nil {
		return model.User{}, err
	}
	if u.LastLoginAt, err = parseNullTime(us.lastLoginAt); err != nil {
		return model.User{}, err
	}
	if u.CreatedAt, err = parseTime(us.createdAt); err != nil {
		return model.User{}, err
	}
	if u.UpdatedAt, err = parseTime(us.updatedAt); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func scanUser(r rowScanner) (model.User, error) {
	var us userScan
	if err := r.Scan(us.dest()...); err != nil {
		return model.User{}, err
	}
	return us.user()
}

func userByIDQ(ctx context.Context, q querier, id int64) (model.User, error) {
	u, err := scanUser(q.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users u WHERE u.id = :id`, sql.Named("id", id)))
	if err != nil {
		return model.User{}, notFound(err)
	}
	return u, nil
}

// CountUsers returns the number of accounts.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CreateUser creates an account and its preferences row in one transaction.
// A username that differs only in case from an existing one is a
// *model.DuplicateError{Kind: "username"}.
func (s *Store) CreateUser(ctx context.Context, n model.NewUser) (model.User, error) {
	key := model.UsernameKey(n.Username)
	locale := n.Locale
	if locale == "" {
		locale = model.DefaultLocale
	}
	var out model.User
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		var existing int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE username_key = :key`,
			sql.Named("key", key)).Scan(&existing)
		if err == nil {
			return &model.DuplicateError{Kind: model.DuplicateUsername}
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check username: %w", err)
		}
		now := formatTime(s.now())
		res, err := tx.ExecContext(ctx, `INSERT INTO users (username, username_key, display_name, role,
				password_hash, must_change_password, password_changed_at, created_at, updated_at)
			VALUES (:username, :key, :display, :role, :hash, :must, :now, :now, :now)`,
			sql.Named("username", n.Username), sql.Named("key", key),
			sql.Named("display", n.DisplayName), sql.Named("role", string(n.Role)),
			sql.Named("hash", n.PasswordHash), sql.Named("must", n.MustChangePassword),
			sql.Named("now", now))
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_preferences (user_id, locale, updated_at)
			VALUES (:id, :locale, :now)`,
			sql.Named("id", id), sql.Named("locale", string(locale)), sql.Named("now", now)); err != nil {
			return fmt.Errorf("insert preferences: %w", err)
		}
		out, err = userByIDQ(ctx, tx, id)
		return err
	})
	if err != nil {
		return model.User{}, err
	}
	return out, nil
}

// UserByID returns an account; model.ErrNotFound when it does not exist.
func (s *Store) UserByID(ctx context.Context, id int64) (model.User, error) {
	return userByIDQ(ctx, s.db, id)
}

// Credentials returns the sign-in view of the account with the given
// username key; model.ErrNotFound when there is none.
func (s *Store) Credentials(ctx context.Context, usernameKey string) (model.Credentials, error) {
	var us userScan
	var c model.Credentials
	var secret sql.NullString
	dest := append(us.dest(), &c.PasswordHash, &secret, &c.TOTPLastStep)
	err := s.db.QueryRowContext(ctx, `SELECT `+userColumns+`, u.password_hash, u.totp_secret, u.totp_last_step
		FROM users u WHERE u.username_key = :key`, sql.Named("key", usernameKey)).Scan(dest...)
	if err != nil {
		return model.Credentials{}, notFound(err)
	}
	if c.User, err = us.user(); err != nil {
		return model.Credentials{}, err
	}
	c.TOTPSecret = secret.String
	return c, nil
}

// adminUserQuery selects a model.AdminUser (scanAdminUser). :now is the
// current time (lock and session expiry).
const adminUserQuery = `SELECT ` + userColumns + `,
	(SELECT MAX(lt.locked_until) FROM login_throttle lt
	  WHERE lt.user_id = u.id AND lt.locked_until > :now) AS locked_until,
	(SELECT COUNT(*) FROM links l WHERE l.owner_id = u.id) AS link_count,
	(SELECT COUNT(*) FROM links l WHERE l.owner_id = u.id
	  AND EXISTS (SELECT 1 FROM link_shares s WHERE s.link_id = l.id)) AS shared_link_count,
	(SELECT COUNT(*) FROM sessions se WHERE se.user_id = u.id AND se.expires_at > :now) AS session_count`

func scanAdminUser(r rowScanner, extra ...any) (model.AdminUser, error) {
	var us userScan
	var locked sql.NullString
	var a model.AdminUser
	dest := append(us.dest(), &locked, &a.LinkCount, &a.SharedLinkCount, &a.SessionCount)
	if err := r.Scan(append(dest, extra...)...); err != nil {
		return model.AdminUser{}, err
	}
	u, err := us.user()
	if err != nil {
		return model.AdminUser{}, err
	}
	a.ID, a.Username, a.DisplayName, a.Role = u.ID, u.Username, u.DisplayName, u.Role
	a.Active, a.MustChangePassword, a.TOTPEnabled = u.Active, u.MustChangePassword, u.TOTPEnabled
	a.LastLoginAt, a.CreatedAt, a.UpdatedAt = u.LastLoginAt, u.CreatedAt, u.UpdatedAt
	if a.LockedUntil, err = parseNullTime(locked); err != nil {
		return model.AdminUser{}, err
	}
	return a, nil
}

// adminUserFilter is the WHERE clause of the admin user list (5.12).
const adminUserFilter = `
	WHERE (:q IS NULL OR instr(u.username_key, :q) > 0 OR instr(` + unicodeLowerFunc + `(u.display_name), :q) > 0)
	  AND (:role IS NULL OR u.role = :role)
	  AND (:status = 'all'
	       OR (:status = 'active' AND u.is_active = 1)
	       OR (:status = 'inactive' AND u.is_active = 0))`

// AdminUsers returns one page of the admin user list ordered by username,
// and the size of the filtered set.
func (s *Store) AdminUsers(ctx context.Context, f model.UserFilter, p model.PageReq) ([]model.AdminUser, int, error) {
	args := []any{
		sql.Named("now", formatTime(s.now())),
		sql.Named("q", nullText(strings.ToLower(strings.TrimSpace(f.Query)))),
		sql.Named("role", nullText(string(f.Role))),
		sql.Named("status", userStatus(f.Status)),
		sql.Named("limit", p.Limit), sql.Named("offset", p.Offset),
	}
	rows, err := s.db.QueryContext(ctx, adminUserQuery+`, COUNT(*) OVER () AS total
		FROM users u`+adminUserFilter+`
		ORDER BY u.username_key, u.id
		LIMIT :limit OFFSET :offset`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	items := []model.AdminUser{}
	total := 0
	for rows.Next() {
		a, err := scanAdminUser(rows, &total)
		if err != nil {
			_ = rows.Close()
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		items = append(items, a)
	}
	if err := closeRows(rows); err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	if len(items) == 0 {
		// Past the last page the window count has no row to ride on.
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u`+adminUserFilter,
			args...).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count users: %w", err)
		}
	}
	return items, total, nil
}

func userStatus(s string) string {
	switch s {
	case model.UserStatusActive, model.UserStatusInactive:
		return s
	}
	return model.UserStatusAll
}

// AdminUser returns one row of the admin user list; model.ErrNotFound when
// the account does not exist.
func (s *Store) AdminUser(ctx context.Context, id int64) (model.AdminUser, error) {
	a, err := scanAdminUser(s.db.QueryRowContext(ctx, adminUserQuery+` FROM users u WHERE u.id = :id`,
		sql.Named("now", formatTime(s.now())), sql.Named("id", id)))
	if err != nil {
		return model.AdminUser{}, notFound(err)
	}
	return a, nil
}

// otherActiveAdmins counts the active admins other than id.
func otherActiveAdmins(ctx context.Context, q querier, id int64) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM users
		WHERE role = 'admin' AND is_active = 1 AND id <> :id`, sql.Named("id", id)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

// UpdateUser changes the display name, role or active flag. Demoting or
// deactivating the last active admin is model.ErrLastAdmin. On a role change
// the links that become visible through a share with the new role start
// their inactivity period at the change (roleVisibilityUpsert), and when an
// owner gets a sharing role back, so do the recipients of the owner's
// returning shares (ownerSharesReturnUpsert).
func (s *Store) UpdateUser(ctx context.Context, id int64, p model.UserPatch) (model.User, error) {
	var out model.User
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		cur, err := userByIDQ(ctx, tx, id)
		if err != nil {
			return err
		}
		role, active, display := cur.Role, cur.Active, cur.DisplayName
		if p.Role != nil {
			role = *p.Role
		}
		if p.Active != nil {
			active = *p.Active
		}
		if p.DisplayName != nil {
			display = *p.DisplayName
		}
		if cur.Role == model.RoleAdmin && cur.Active && (role != model.RoleAdmin || !active) {
			n, err := otherActiveAdmins(ctx, tx, id)
			if err != nil {
				return err
			}
			if n == 0 {
				return model.ErrLastAdmin
			}
		}
		if p.Role != nil || p.Active != nil || p.DisplayName != nil {
			_, err = tx.ExecContext(ctx, `UPDATE users SET role = :role, is_active = :active,
				display_name = :display, updated_at = :now WHERE id = :id`,
				sql.Named("role", string(role)), sql.Named("active", active),
				sql.Named("display", display), sql.Named("now", formatTime(s.now())),
				sql.Named("id", id))
			if err != nil {
				return fmt.Errorf("update user: %w", err)
			}
		}
		if role != cur.Role {
			// Links shared with the new role become visible now (7.2).
			if _, err := tx.ExecContext(ctx, roleVisibilityUpsert,
				sql.Named("uid", id), sql.Named("role", string(role)),
				sql.Named("old_role", string(cur.Role)), sql.Named("now", formatTime(s.now()))); err != nil {
				return fmt.Errorf("start visibility of role shares: %w", err)
			}
		}
		if !cur.Role.CanShare() && role.CanShare() {
			// The owner's dormant shares become visible again (4.7, 7.2).
			if _, err := tx.ExecContext(ctx, ownerSharesReturnUpsert,
				sql.Named("uid", id), sql.Named("now", formatTime(s.now()))); err != nil {
				return fmt.Errorf("restart visibility of the owner's shares: %w", err)
			}
		}
		out, err = userByIDQ(ctx, tx, id)
		return err
	})
	if err != nil {
		return model.User{}, err
	}
	return out, nil
}

// SetPassword stores a new password hash and the must-change flag.
func (s *Store) SetPassword(ctx context.Context, id int64, hash string, mustChange bool) error {
	now := formatTime(s.now())
	res, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = :hash,
		must_change_password = :must, password_changed_at = :now, updated_at = :now WHERE id = :id`,
		sql.Named("hash", hash), sql.Named("must", mustChange), sql.Named("now", now),
		sql.Named("id", id))
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	return affectedOne(res)
}

// RecordLogin stores the time of a successful sign-in.
func (s *Store) RecordLogin(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET last_login_at = :now WHERE id = :id`,
		sql.Named("now", formatTime(s.now())), sql.Named("id", id))
	if err != nil {
		return fmt.Errorf("record login: %w", err)
	}
	return affectedOne(res)
}

// Directory lists active users other than exclude whose username or display
// name contains q (case-insensitive), ordered by username (5.11).
func (s *Store) Directory(ctx context.Context, exclude int64, q string, limit int) ([]model.DirectoryUser, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT u.id, u.username, u.display_name, u.role
		FROM users u
		WHERE u.is_active = 1 AND u.id <> :exclude
		  AND (:q IS NULL OR instr(u.username_key, :q) > 0
		       OR instr(`+unicodeLowerFunc+`(u.display_name), :q) > 0)
		ORDER BY u.username_key, u.id
		LIMIT :limit`,
		sql.Named("exclude", exclude),
		sql.Named("q", nullText(strings.ToLower(strings.TrimSpace(q)))),
		sql.Named("limit", limit))
	if err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}
	items := []model.DirectoryUser{}
	for rows.Next() {
		var d model.DirectoryUser
		var role string
		if err := rows.Scan(&d.ID, &d.Username, &d.DisplayName, &role); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan directory: %w", err)
		}
		d.Role = model.Role(role)
		items = append(items, d)
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}
	return items, nil
}

// DeleteUser deletes an account (6.11): in one transaction it checks the last
// admin rule, appends the given audit entry, pseudonymises every audit row
// that names the user (the new one included) and deletes the user, whose
// rows cascade. The database runs with secure_delete, and the WAL is
// truncated afterwards, so the erased data does not stay readable in the
// files of the data directory.
func (s *Store) DeleteUser(ctx context.Context, id int64, e model.AuditEntry) error {
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		u, err := userByIDQ(ctx, tx, id)
		if err != nil {
			return err
		}
		if u.Role == model.RoleAdmin && u.Active {
			n, err := otherActiveAdmins(ctx, tx, id)
			if err != nil {
				return err
			}
			if n == 0 {
				return model.ErrLastAdmin
			}
		}
		if err := appendAuditQ(ctx, tx, e, s.now()); err != nil {
			return err
		}
		alias := fmt.Sprintf("deleted-user-%d", id)
		if _, err := tx.ExecContext(ctx, `UPDATE audit_log SET actor_name = :alias
			WHERE actor_id = :id OR (actor_id IS NULL AND lower(actor_name) = :key)`,
			sql.Named("alias", alias), sql.Named("id", id), sql.Named("key", u.UsernameKey)); err != nil {
			return fmt.Errorf("pseudonymise audit actor: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE audit_log SET target_name = :alias
			WHERE target_type = 'user' AND target_id = :id`,
			sql.Named("alias", alias), sql.Named("id", id)); err != nil {
			return fmt.Errorf("pseudonymise audit target: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = :id`, sql.Named("id", id)); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.truncateWAL(ctx)
	return nil
}

// truncateWAL copies the write-ahead log into app.db and truncates it, so
// the page images that still hold erased data leave the WAL file at once
// (secure_delete has already zeroed them in the new page versions). It is
// best effort: the deletion has committed, and a checkpoint that cannot
// finish now happens automatically later.
func (s *Store) truncateWAL(ctx context.Context) {
	_, _ = s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
}

// OwnSharedLinkCount counts the user's own links that have at least one share.
func (s *Store) OwnSharedLinkCount(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM links l WHERE l.owner_id = :uid
		AND EXISTS (SELECT 1 FROM link_shares s WHERE s.link_id = l.id)`,
		sql.Named("uid", userID)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count shared links: %w", err)
	}
	return n, nil
}

// nullText binds "" as NULL (optional filters, 4.1).
func nullText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// affectedOne returns model.ErrNotFound when an UPDATE or DELETE touched no row.
func affectedOne(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// closeRows closes rows and reports an iteration or close error.
func closeRows(rows *sql.Rows) error {
	iterErr := rows.Err()
	if err := rows.Close(); err != nil {
		return err
	}
	return iterErr
}
