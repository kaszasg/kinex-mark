package model

import "time"

// Viewer identifies the user on whose behalf visibility is evaluated (4.7).
type Viewer struct {
	UserID int64
	Role   Role
}

// TagColor is a key of the fixed tag color palette (4.6).
type TagColor string

// The tag palette in picker order.
const (
	TagColorBlue     TagColor = "blue"
	TagColorOchre    TagColor = "ochre"
	TagColorGreen    TagColor = "green"
	TagColorViolet   TagColor = "violet"
	TagColorRose     TagColor = "rose"
	TagColorNeutral  TagColor = "neutral"
	TagColorContrast TagColor = "contrast"
)

// DefaultTagColor is used when a tag is created without a color.
const DefaultTagColor = TagColorNeutral

// TagColors lists the palette in the order of section 4.6.
var TagColors = []TagColor{
	TagColorBlue, TagColorOchre, TagColorGreen, TagColorViolet,
	TagColorRose, TagColorNeutral, TagColorContrast,
}

// Valid reports whether c is a palette key.
func (c TagColor) Valid() bool {
	for _, k := range TagColors {
		if c == k {
			return true
		}
	}
	return false
}

// FetchStatus is the metadata fetch state of a link.
type FetchStatus string

// Fetch states (links.fetch_status).
const (
	FetchPending FetchStatus = "pending"
	FetchOK      FetchStatus = "ok"
	FetchFailed  FetchStatus = "failed"
	FetchBlocked FetchStatus = "blocked"
)

// TitleSource records where a link title came from (4.4).
type TitleSource string

// Title sources (links.title_source).
const (
	TitleNone    TitleSource = "none"
	TitleFetched TitleSource = "fetched"
	TitleManual  TitleSource = "manual"
)

// ArchiveReason records why a link was archived for a user.
type ArchiveReason string

// Archive reasons (link_user_state.archive_reason).
const (
	ArchiveAuto   ArchiveReason = "auto"
	ArchiveManual ArchiveReason = "manual"
)

// Link limits (4.4, 5.3, 5.9).
const (
	MaxLinksPerUser   = 10_000
	MaxTagsPerLink    = 20
	URLMaxBytes       = 2048
	HostMaxLen        = 253
	TitleMaxRunes     = 500
	DescriptionMaxLen = 500
	QuickTextMaxRunes = 8192
	SearchMaxRunes    = 200
)

// Link list query values (5.9.1).
const (
	ScopeAll    = "all"
	ScopeOwn    = "own"
	ScopeShared = "shared"

	ArchivedExclude = "exclude"
	ArchivedInclude = "include"
	ArchivedOnly    = "only"
)

// Allowed values of the enumerated link query parameters.
var (
	LinkScopes    = []string{ScopeAll, ScopeOwn, ScopeShared}
	ArchivedModes = []string{ArchivedExclude, ArchivedInclude, ArchivedOnly}
)

// LinkItem is a link as seen by one viewer (5.5 LinkItem). The derived JSON
// fields inactiveSince, archiveDueAt and iconUrl are computed only by
// internal/api/dto.go from BaselineAt, LastUsedAt and IconSHA256.
type LinkItem struct {
	ID            int64            `json:"id"`
	URL           string           `json:"url"`
	Host          string           `json:"host"`
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	FetchStatus   FetchStatus      `json:"fetchStatus"`
	TagIDs        []int64          `json:"tagIds"` // the viewer's own tags only
	Pinned        bool             `json:"pinned"`
	UseCount      int64            `json:"useCount"`
	LastUsedAt    *time.Time       `json:"lastUsedAt"`
	ArchivedAt    *time.Time       `json:"archivedAt"`
	ArchiveReason *ArchiveReason   `json:"archiveReason"`
	IsOwner       bool             `json:"isOwner"`
	Owner         UserRef          `json:"owner"`
	ShareCount    int              `json:"shareCount"` // non-zero only for the owner
	SharedVia     *ShareTargetType `json:"sharedVia"`  // recipients only; priority user > role > all
	SharedAt      *time.Time       `json:"sharedAt"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`

	BaselineAt time.Time `json:"-"` // inactivity baseline (4.8)
	IconSHA256 string    `json:"-"` // "" when the link has no icon
}

// LinkDetail is the full view of one link (5.5 LinkDetail).
type LinkDetail struct {
	LinkItem
	TitleSource   TitleSource `json:"titleSource"`
	Tags          []Tag       `json:"tags"` // TagIDs resolved
	FetchError    *string     `json:"fetchError"`
	FetchedAt     *time.Time  `json:"fetchedAt"`
	FetchAttempts int         `json:"fetchAttempts"`
	Shares        []Share     `json:"shares"` // owner with share permission only, otherwise empty
}

// LinkState is the personal state of a viewer on a link (5.5 LinkState).
type LinkState struct {
	Pinned        bool           `json:"pinned"`
	UseCount      int64          `json:"useCount"`
	LastUsedAt    *time.Time     `json:"lastUsedAt"`
	ArchivedAt    *time.Time     `json:"archivedAt"`
	ArchiveReason *ArchiveReason `json:"archiveReason"`

	BaselineAt time.Time `json:"-"`
}

// LinkQuery is the validated GET /api/links query (5.9.1, 4.8).
type LinkQuery struct {
	Q        string  // trimmed and lower-cased; "" for none
	TagIDs   []int64 // empty for none
	TagMatch string  // TagMatchAny or TagMatchAll
	Untagged bool
	Scope    string // ScopeAll, ScopeOwn or ScopeShared
	Archived string // ArchivedExclude, ArchivedInclude or ArchivedOnly
	Sort     string // SortUsage, SortCreated or SortTitle
	Limit    int
	Offset   int
}

// LinkCounts are counts within the filtered link set (4.8).
type LinkCounts struct {
	Pinned   int `json:"pinned"`
	Archived int `json:"archived"`
}

// LinkPage is one page of the link list (5.5 LinkPage).
type LinkPage struct {
	Items  []LinkItem `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	Counts LinkCounts `json:"counts"`
}

// NewLink is the input of Store.CreateLink. URL and Host come from
// fetch.NormalizeURL; a non-empty Title is a manual title (4.4).
type NewLink struct {
	OwnerID     int64
	URL         string
	Host        string
	Title       string
	Description string
	TagIDs      []int64
	Pinned      bool
}

// LinkPatch is the input of Store.UpdateLink; nil fields stay unchanged.
// URL and Host are set together (fetch.NormalizeURL output).
type LinkPatch struct {
	URL         *string
	Host        *string
	Title       *string
	Description *string
}

// IconBlob is a stored favicon.
type IconBlob struct {
	Data        []byte
	ContentType string
	SHA256      string // lower-case hex
}

// Icon content types allowed by the icons table (4.4).
var IconContentTypes = []string{
	"image/png", "image/x-icon", "image/gif", "image/jpeg", "image/webp", "image/svg+xml",
}

// MaxIconBytes is the icon size limit (icons.data CHECK, 6.8).
const MaxIconBytes = 64 << 10
