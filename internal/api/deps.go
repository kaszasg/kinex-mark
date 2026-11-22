package api

import (
	"context"
	"io/fs"
	"log/slog"
	"time"

	"github.com/kaszasg/kinex-mark/internal/auth"
	"github.com/kaszasg/kinex-mark/internal/clock"
	"github.com/kaszasg/kinex-mark/internal/config"
	"github.com/kaszasg/kinex-mark/internal/model"
)

// Deps are the dependencies of the HTTP handler (3.3.10).
type Deps struct {
	Store   Store
	Jobs    Jobs
	Hasher  *auth.Hasher
	Keyring *auth.Keyring
	Clock   clock.Clock
	Log     *slog.Logger
	Config  config.Config
	Version string
	UI      fs.FS // the embedded SvelteKit build (web.Files)
}

// Jobs is the part of jobs.Runner the handlers use.
type Jobs interface {
	// Enqueue queues a metadata fetch without blocking; false when the queue
	// is full (the sweeper picks the link up later, 7.3).
	Enqueue(linkID int64) bool
	// RunArchive runs the automatic archiving job now (POST /api/admin/jobs/archive).
	RunArchive(ctx context.Context) (int, error)
}

// Store is the part of store.Store the handlers use. The api package never
// imports the store; internal/app asserts the implementation at compile time.
type Store interface {
	Ping(ctx context.Context) error

	// settings
	Settings(ctx context.Context) (model.SystemSettings, error)
	UpdateSettings(ctx context.Context, p model.SettingsPatch, by int64) (model.SystemSettings, error)
	BumpPrivacyNotice(ctx context.Context, by int64) (model.SystemSettings, error)

	// users
	CreateUser(ctx context.Context, u model.NewUser) (model.User, error)
	UserByID(ctx context.Context, id int64) (model.User, error)
	Credentials(ctx context.Context, usernameKey string) (model.Credentials, error)
	AdminUsers(ctx context.Context, f model.UserFilter, p model.PageReq) ([]model.AdminUser, int, error)
	AdminUser(ctx context.Context, id int64) (model.AdminUser, error)
	UpdateUser(ctx context.Context, id int64, p model.UserPatch) (model.User, error)
	SetPassword(ctx context.Context, id int64, hash string, mustChange bool) error
	RecordLogin(ctx context.Context, id int64) error
	Directory(ctx context.Context, exclude int64, q string, limit int) ([]model.DirectoryUser, error)
	DeleteUser(ctx context.Context, id int64, e model.AuditEntry) error
	OwnSharedLinkCount(ctx context.Context, userID int64) (int, error)

	// totp
	SetPendingTOTP(ctx context.Context, id int64, sealed string, expires time.Time) error
	PendingTOTP(ctx context.Context, id int64) (sealed string, expires time.Time, err error)
	EnableTOTP(ctx context.Context, id int64, sealed string, step int64, codeHashes []string) error
	DisableTOTP(ctx context.Context, id int64) error
	AdvanceTOTPStep(ctx context.Context, id, step int64) (bool, error)
	ReplaceRecoveryCodes(ctx context.Context, id int64, codeHashes []string) error
	UseRecoveryCode(ctx context.Context, id int64, codeHash string) (bool, error)
	RecoveryCodesRemaining(ctx context.Context, id int64) (int, error)

	// sessions, challenges
	CreateSession(ctx context.Context, n model.NewSession) (model.Session, error)
	SessionByToken(ctx context.Context, hash []byte) (model.Session, model.User, error)
	TouchSession(ctx context.Context, id int64) error
	Sessions(ctx context.Context, userID int64) ([]model.Session, error)
	DeleteSession(ctx context.Context, userID, id int64) error
	DeleteSessionByToken(ctx context.Context, hash []byte) error
	DeleteUserSessions(ctx context.Context, userID, exceptID int64) (int, error)
	CreateChallenge(ctx context.Context, hash []byte, userID int64, ip string, expires time.Time) error
	ChallengeAttempt(ctx context.Context, hash []byte, maxAttempts int) (model.Challenge, error)
	DeleteChallenge(ctx context.Context, hash []byte) error

	// throttle
	ThrottleUpdate(ctx context.Context, keys []model.ThrottleKey,
		fn func(row model.ThrottleRow) (model.ThrottleRow, error)) error
	ClearThrottle(ctx context.Context, key string) error
	ClearUserThrottle(ctx context.Context, userID int64) error

	// preferences
	Preferences(ctx context.Context, userID int64) (model.Preferences, error)
	UpdatePreferences(ctx context.Context, userID int64, p model.PreferencesPatch) (model.Preferences, error)

	// links
	CreateLink(ctx context.Context, n model.NewLink) (model.LinkDetail, error)
	Link(ctx context.Context, v model.Viewer, id int64) (model.LinkDetail, error)
	Links(ctx context.Context, v model.Viewer, q model.LinkQuery) (model.LinkPage, error)
	LinkItem(ctx context.Context, v model.Viewer, id int64) (model.LinkItem, error)
	UpdateLink(ctx context.Context, v model.Viewer, id int64, p model.LinkPatch) (model.LinkDetail, error)
	DeleteLink(ctx context.Context, v model.Viewer, id int64) error
	SetLinkTags(ctx context.Context, v model.Viewer, id int64, tagIDs []int64) error
	SetPinned(ctx context.Context, v model.Viewer, id int64, pinned bool) error
	SetArchived(ctx context.Context, v model.Viewer, id int64, archived bool) error
	RecordVisit(ctx context.Context, v model.Viewer, id int64) (model.LinkState, error)
	LinkIcon(ctx context.Context, v model.Viewer, id int64) (model.IconBlob, error)
	ResetFetch(ctx context.Context, v model.Viewer, id int64) error

	// shares, tags
	Shares(ctx context.Context, v model.Viewer, linkID int64) ([]model.Share, error)
	CreateShare(ctx context.Context, v model.Viewer, linkID int64, t model.ShareTarget) (model.Share, error)
	DeleteShare(ctx context.Context, v model.Viewer, linkID, shareID int64) error
	Tags(ctx context.Context, userID int64, role model.Role) ([]model.Tag, error)
	CreateTag(ctx context.Context, userID int64, name string, color model.TagColor) (model.Tag, error)
	UpdateTag(ctx context.Context, userID, id int64, p model.TagPatch) (model.Tag, error)
	DeleteTag(ctx context.Context, userID, id int64) error

	// audit, consent, gdpr
	AppendAudit(ctx context.Context, e model.AuditEntry) error
	Audit(ctx context.Context, f model.AuditFilter, p model.PageReq) ([]model.AuditEntry, int, error)
	RecordConsent(ctx context.Context, userID int64, kind string, version int) error
	Consents(ctx context.Context, userID int64) ([]model.Consent, error)
	ExportUser(ctx context.Context, userID int64) (model.UserExport, error)
	ExportLinks(ctx context.Context, ownerID int64) ([]model.ExportedLink, error)
	ImportLinks(ctx context.Context, ownerID int64, entries []model.ImportEntry) (model.ImportResult, []int64, error)
}
