package model

import "time"

// ShareTargetType is the audience of a link share.
type ShareTargetType string

// Share audiences (link_shares.target_type).
const (
	ShareAll  ShareTargetType = "all"
	ShareRole ShareTargetType = "role"
	ShareUser ShareTargetType = "user"
)

// ShareTargetTypes lists the share audiences.
var ShareTargetTypes = []ShareTargetType{ShareAll, ShareRole, ShareUser}

// Valid reports whether t is a known share audience.
func (t ShareTargetType) Valid() bool {
	return t == ShareAll || t == ShareRole || t == ShareUser
}

// Share is a link share (5.5 Share).
type Share struct {
	ID         int64           `json:"id"`
	TargetType ShareTargetType `json:"targetType"`
	TargetRole *Role           `json:"targetRole"`
	TargetUser *UserRef        `json:"targetUser"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// ShareTarget is the input of a new share (5.5 ShareTarget). For ShareRole
// only TargetRole is set, for ShareUser only TargetUserID, for ShareAll neither.
type ShareTarget struct {
	TargetType   ShareTargetType `json:"targetType"`
	TargetRole   *Role           `json:"targetRole,omitempty"`
	TargetUserID *int64          `json:"targetUserId,omitempty"`
}
