// Package model holds the domain types, sentinel errors and constants shared
// by the store, auth, jobs and api packages. It contains no I/O.
package model

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by the store. The api package maps them to HTTP
// error codes (internal/api/errors.go).
var (
	// ErrNotFound: the row does not exist or is not visible to the viewer.
	ErrNotFound = errors.New("not found")
	// ErrNotOwner: the link is visible, but the operation is reserved for its owner.
	ErrNotOwner = errors.New("not owner")
	// ErrConflict: the row changed state so the operation cannot be applied.
	ErrConflict = errors.New("conflict")
	// ErrLastAdmin: the operation would leave the system without an active admin.
	ErrLastAdmin = errors.New("last active admin")
	// ErrLimitReached: the per-user link or tag limit is reached.
	ErrLimitReached = errors.New("limit reached")
)

// Duplicate kinds reported by DuplicateError.
const (
	DuplicateLink     = "link"
	DuplicateTag      = "tag"
	DuplicateShare    = "share"
	DuplicateUsername = "username"
)

// DuplicateError reports a uniqueness violation. ExistingID is the id of the
// existing row when known (link and tag), otherwise 0.
type DuplicateError struct {
	Kind       string // DuplicateLink, DuplicateTag, DuplicateShare or DuplicateUsername
	ExistingID int64
}

// Error implements the error interface.
func (e *DuplicateError) Error() string {
	if e.ExistingID > 0 {
		return fmt.Sprintf("duplicate %s (existing id %d)", e.Kind, e.ExistingID)
	}
	return "duplicate " + e.Kind
}
