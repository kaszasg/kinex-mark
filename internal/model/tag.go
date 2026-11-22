package model

import "time"

// Tag limits (4.4, 5.10).
const (
	MaxTagsPerUser = 500
	TagNameMaxLen  = 40
)

// TagNameKey returns the per-owner uniqueness key of a tag name (tags.name_key).
func TagNameKey(name string) string {
	return UsernameKey(name)
}

// Tag is a personal tag (5.5 Tag).
type Tag struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Color           TagColor  `json:"color"`
	LinkCount       int       `json:"linkCount"`       // visible links carrying this tag
	ActiveLinkCount int       `json:"activeLinkCount"` // of which not archived
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// TagPatch is the input of Store.UpdateTag; nil fields stay unchanged.
type TagPatch struct {
	Name  *string
	Color *TagColor
}
