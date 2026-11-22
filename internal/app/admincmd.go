package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
)

var (
	errNoAdminCommand = errors.New("no admin command given")
	errUnknownUser    = errors.New("user not found")
	errUnknownCommand = errors.New("unknown admin command")
)

// adminCommandFailed is the log message of a failed recovery command.
const adminCommandFailed = "admin command failed"

// RunAdminCommand executes an offline recovery command (-unlock-user,
// -reset-password, 6.12) and returns the exit code: 0 on success, 1 on
// failure such as an unknown user. It opens the data directory, the key ring
// and the store like Run, but starts neither the HTTP server nor the jobs.
// A generated password goes to out only, never to the log.
func RunAdminCommand(ctx context.Context, cfg config.Config, out io.Writer, log *slog.Logger) int {
	if cfg.AdminCommand == nil {
		log.Error(adminCommandFailed, "err", errNoAdminCommand)
		return 1
	}
	cmd := *cfg.AdminCommand
	st, _, err := openData(ctx, cfg, log)
	if err != nil {
		log.Error(adminCommandFailed, "command", cmd.Kind, "err", err)
		return 1
	}
	defer st.Close()

	h := auth.NewHasher(hasherIterations, hasherConcurrency)
	if err := runAdminCommand(ctx, st, h, cmd, out); err != nil {
		log.Error(adminCommandFailed, "command", cmd.Kind, "err", err)
		return 1
	}
	log.Info("admin command completed", "command", cmd.Kind, "username", cmd.Username)
	return 0
}

// adminStore is the part of the store that the recovery commands use.
type adminStore interface {
	Credentials(ctx context.Context, usernameKey string) (model.Credentials, error)
	ClearUserThrottle(ctx context.Context, userID int64) error
	SetPassword(ctx context.Context, id int64, hash string, mustChange bool) error
	DeleteUserSessions(ctx context.Context, userID, exceptID int64) (int, error)
	AppendAudit(ctx context.Context, e model.AuditEntry) error
}

// runAdminCommand performs one recovery command against st.
func runAdminCommand(ctx context.Context, st adminStore, h *auth.Hasher, cmd config.AdminCommand, out io.Writer) error {
	creds, err := st.Credentials(ctx, model.UsernameKey(cmd.Username))
	if errors.Is(err, model.ErrNotFound) {
		return errUnknownUser
	}
	if err != nil {
		return fmt.Errorf("load user: %w", err)
	}
	u := creds.User

	switch cmd.Kind {
	case config.AdminCommandUnlockUser:
		if err := st.ClearUserThrottle(ctx, u.ID); err != nil {
			return fmt.Errorf("clear throttle: %w", err)
		}
		if err := st.AppendAudit(ctx, cliAudit(model.EventUserUnlocked, u)); err != nil {
			return fmt.Errorf("audit: %w", err)
		}
		_, err = fmt.Fprintf(out, "Sign-in lock cleared for user %s.\n", u.Username)
		return err

	case config.AdminCommandResetPassword:
		password, err := auth.GeneratePassword(generatedPasswordLength)
		if err != nil {
			return fmt.Errorf("generate password: %w", err)
		}
		hash, err := h.Hash(password)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		if err := st.SetPassword(ctx, u.ID, hash, true); err != nil {
			return fmt.Errorf("set password: %w", err)
		}
		if _, err := st.DeleteUserSessions(ctx, u.ID, 0); err != nil {
			return fmt.Errorf("delete sessions: %w", err)
		}
		if err := st.ClearUserThrottle(ctx, u.ID); err != nil {
			return fmt.Errorf("clear throttle: %w", err)
		}
		if err := st.AppendAudit(ctx, cliAudit(model.EventUserPasswordReset, u)); err != nil {
			return fmt.Errorf("audit: %w", err)
		}
		_, err = fmt.Fprintf(out, "New password for user %s: %s\nThe password must be changed at the next sign-in.\n", u.Username, password)
		return err

	default:
		return fmt.Errorf("%w: %q", errUnknownCommand, cmd.Kind)
	}
}

// cliAudit builds the audit entry of a recovery command: no actor, the user
// as target and details {via: "cli"} (6.12).
func cliAudit(event string, u model.User) model.AuditEntry {
	return model.AuditEntry{
		Event:   event,
		Success: true,
		Target:  &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: u.Username},
		Details: map[string]any{model.DetailVia: model.ViaCLI},
	}
}
