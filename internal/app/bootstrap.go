package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// generatedPasswordLength is the length of generated admin passwords (6.12).
const generatedPasswordLength = 20

// errWeakAdminPassword stops the start when APP_ADMIN_PASSWORD breaks the
// password policy (6.12).
var errWeakAdminPassword = errors.New("bootstrap admin: password does not meet the policy")

// bootstrapStore is the part of the store that bootstrapAdmin uses.
type bootstrapStore interface {
	CountUsers(ctx context.Context) (int, error)
	CreateUser(ctx context.Context, n model.NewUser) (model.User, error)
	AppendAudit(ctx context.Context, e model.AuditEntry) error
}

// bootstrapAdmin creates the first admin on an empty database (6.12). With
// APP_ADMIN_PASSWORD the password must pass the policy and need not be
// changed; without it a generated password is logged once and must be
// changed at the first sign-in. On a non-empty database a set
// APP_ADMIN_PASSWORD only produces a warning.
func bootstrapAdmin(ctx context.Context, st bootstrapStore, h *auth.Hasher, cfg config.Config, log *slog.Logger) error {
	n, err := st.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap admin: count users: %w", err)
	}
	if n > 0 {
		if cfg.AdminPassword != "" {
			log.Warn("APP_ADMIN_PASSWORD is set but ignored on a non-empty database; remove it")
		}
		return nil
	}

	username := cfg.AdminUser
	password := cfg.AdminPassword
	generated := password == ""
	if generated {
		password, err = auth.GeneratePassword(generatedPasswordLength)
		if err != nil {
			return fmt.Errorf("bootstrap admin: generate password: %w", err)
		}
	} else if auth.CheckPasswordPolicy(password, username, "") != "" {
		return errWeakAdminPassword
	}
	hash, err := h.Hash(password)
	if err != nil {
		return fmt.Errorf("bootstrap admin: hash password: %w", err)
	}
	u, err := st.CreateUser(ctx, model.NewUser{
		Username:           username,
		Role:               model.RoleAdmin,
		PasswordHash:       hash,
		MustChangePassword: generated,
	})
	if err != nil {
		return fmt.Errorf("bootstrap admin: create user: %w", err)
	}
	err = st.AppendAudit(ctx, model.AuditEntry{
		Event:   model.EventBootstrapAdminCreated,
		Success: true,
		Target:  &model.AuditTarget{Type: model.TargetUser, ID: &u.ID, Name: u.Username},
		Details: map[string]any{model.DetailGeneratedPassword: generated},
	})
	if err != nil {
		return fmt.Errorf("bootstrap admin: audit: %w", err)
	}
	if generated {
		// The only password that is ever logged (6.12).
		log.Warn("initial admin password generated", "username", u.Username, "password", password)
	} else {
		log.Warn("bootstrap admin created from APP_ADMIN_PASSWORD; remove the variable from the environment", "username", u.Username)
	}
	return nil
}
