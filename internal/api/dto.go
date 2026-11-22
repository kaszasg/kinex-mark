package api

import (
	"strconv"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// iconVersionLen is the number of hex characters of the icon SHA-256 used as
// the cache buster of iconUrl (5.5).
const iconVersionLen = 8

// LinkItem is the response shape of a link as seen by one viewer (5.5
// LinkItem). Unlike model.LinkItem it carries the derived fields iconUrl,
// inactiveSince and archiveDueAt, which only toLinkItem computes.
type LinkItem struct {
	ID            int64                  `json:"id"`
	URL           string                 `json:"url"`
	Host          string                 `json:"host"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	IconURL       *string                `json:"iconUrl"`
	FetchStatus   model.FetchStatus      `json:"fetchStatus"`
	TagIDs        []int64                `json:"tagIds"`
	Pinned        bool                   `json:"pinned"`
	UseCount      int64                  `json:"useCount"`
	LastUsedAt    *time.Time             `json:"lastUsedAt"`
	ArchivedAt    *time.Time             `json:"archivedAt"`
	ArchiveReason *model.ArchiveReason   `json:"archiveReason"`
	InactiveSince time.Time              `json:"inactiveSince"`
	ArchiveDueAt  *time.Time             `json:"archiveDueAt"`
	IsOwner       bool                   `json:"isOwner"`
	Owner         model.UserRef          `json:"owner"`
	ShareCount    int                    `json:"shareCount"`
	SharedVia     *model.ShareTargetType `json:"sharedVia"`
	SharedAt      *time.Time             `json:"sharedAt"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// LinkDetail is the full view of one link (5.5 LinkDetail).
type LinkDetail struct {
	LinkItem
	TitleSource   model.TitleSource `json:"titleSource"`
	Tags          []model.Tag       `json:"tags"`
	FetchError    *string           `json:"fetchError"`
	FetchedAt     *time.Time        `json:"fetchedAt"`
	FetchAttempts int               `json:"fetchAttempts"`
	Shares        []model.Share     `json:"shares"`
}

// LinkState is the personal state of the viewer on a link (5.5 LinkState).
type LinkState struct {
	Pinned        bool                 `json:"pinned"`
	UseCount      int64                `json:"useCount"`
	LastUsedAt    *time.Time           `json:"lastUsedAt"`
	ArchivedAt    *time.Time           `json:"archivedAt"`
	ArchiveReason *model.ArchiveReason `json:"archiveReason"`
	InactiveSince time.Time            `json:"inactiveSince"`
	ArchiveDueAt  *time.Time           `json:"archiveDueAt"`
}

// LinkPage is one page of the link list (5.5 LinkPage).
type LinkPage struct {
	Items  []LinkItem       `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Counts model.LinkCounts `json:"counts"`
}

// QuickCaptureResult is the answer of POST /api/links/quick (5.5).
type QuickCaptureResult struct {
	Link    LinkDetail `json:"link"`
	Created bool       `json:"created"`
}

// toLinkItem converts a store row into the response shape and computes the
// derived fields (5.5): inactiveSince = max(BaselineAt, LastUsedAt);
// archiveDueAt = inactiveSince + archiveDays, or null when the link is pinned
// or archived or archiveDays is 0 (never); iconUrl only when the link has an
// icon. archiveDays is the viewer's effective archive period
// (Preferences.EffectiveArchiveAfterDays).
func toLinkItem(m model.LinkItem, archiveDays int) LinkItem {
	inactive := inactiveSince(m.BaselineAt, m.LastUsedAt)
	return LinkItem{
		ID:            m.ID,
		URL:           m.URL,
		Host:          m.Host,
		Title:         m.Title,
		Description:   m.Description,
		IconURL:       iconURL(m.ID, m.IconSHA256),
		FetchStatus:   m.FetchStatus,
		TagIDs:        nonNil(m.TagIDs),
		Pinned:        m.Pinned,
		UseCount:      m.UseCount,
		LastUsedAt:    utcPtr(m.LastUsedAt),
		ArchivedAt:    utcPtr(m.ArchivedAt),
		ArchiveReason: m.ArchiveReason,
		InactiveSince: inactive,
		ArchiveDueAt:  archiveDueAt(inactive, m.Pinned, m.ArchivedAt != nil, archiveDays),
		IsOwner:       m.IsOwner,
		Owner:         m.Owner,
		ShareCount:    m.ShareCount,
		SharedVia:     m.SharedVia,
		SharedAt:      utcPtr(m.SharedAt),
		CreatedAt:     m.CreatedAt.UTC(),
		UpdatedAt:     m.UpdatedAt.UTC(),
	}
}

// toLinkDetail converts a detail row; nil lists become empty ones.
func toLinkDetail(m model.LinkDetail, archiveDays int) LinkDetail {
	return LinkDetail{
		LinkItem:      toLinkItem(m.LinkItem, archiveDays),
		TitleSource:   m.TitleSource,
		Tags:          nonNil(m.Tags),
		FetchError:    m.FetchError,
		FetchedAt:     utcPtr(m.FetchedAt),
		FetchAttempts: m.FetchAttempts,
		Shares:        nonNil(m.Shares),
	}
}

// toLinkState converts a personal state with the same derived fields as
// toLinkItem.
func toLinkState(m model.LinkState, archiveDays int) LinkState {
	inactive := inactiveSince(m.BaselineAt, m.LastUsedAt)
	return LinkState{
		Pinned:        m.Pinned,
		UseCount:      m.UseCount,
		LastUsedAt:    utcPtr(m.LastUsedAt),
		ArchivedAt:    utcPtr(m.ArchivedAt),
		ArchiveReason: m.ArchiveReason,
		InactiveSince: inactive,
		ArchiveDueAt:  archiveDueAt(inactive, m.Pinned, m.ArchivedAt != nil, archiveDays),
	}
}

// toLinkPage converts a list page.
func toLinkPage(p model.LinkPage, req model.LinkQuery, archiveDays int) LinkPage {
	items := make([]LinkItem, 0, len(p.Items))
	for _, it := range p.Items {
		items = append(items, toLinkItem(it, archiveDays))
	}
	return LinkPage{Items: items, Total: p.Total, Limit: req.Limit, Offset: req.Offset, Counts: p.Counts}
}

// inactiveSince is the later of the baseline and the last use, in UTC.
func inactiveSince(baseline time.Time, lastUsed *time.Time) time.Time {
	if lastUsed != nil && lastUsed.After(baseline) {
		return lastUsed.UTC()
	}
	return baseline.UTC()
}

// archiveDueAt is inactive + days, or nil when the link is pinned, archived
// or never archived automatically (days <= 0).
func archiveDueAt(inactive time.Time, pinned, archived bool, days int) *time.Time {
	if pinned || archived || days <= 0 {
		return nil
	}
	due := inactive.AddDate(0, 0, days)
	return &due
}

// iconURL is "/api/links/<id>/icon?v=<first 8 hex of the SHA-256>", or nil
// when the link has no icon.
func iconURL(id int64, sha string) *string {
	if sha == "" {
		return nil
	}
	if len(sha) > iconVersionLen {
		sha = sha[:iconVersionLen]
	}
	u := "/api/links/" + strconv.FormatInt(id, 10) + "/icon?v=" + sha
	return &u
}

// utcPtr returns a copy of t in UTC; nil stays nil.
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
