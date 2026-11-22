package model

import (
	"fmt"
	"time"
)

// TimeLayout is the database timestamp format: always three fractional
// digits and "Z", so that text comparison is chronological (4.1).
const TimeLayout = "2006-01-02T15:04:05.000Z07:00"

// FormatTime formats t in UTC with TimeLayout.
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeLayout)
}

// ParseTime parses a timestamp written by FormatTime. RFC 3339 values with
// any number of fractional digits (for example SQL DEFAULT values) are
// accepted as well. The result is in UTC.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(TimeLayout, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
		}
	}
	return t.UTC(), nil
}

// FormatTimePtr formats a nullable timestamp; nil stays nil.
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatTime(*t)
	return &s
}

// ParseTimePtr parses a nullable timestamp; nil stays nil.
func ParseTimePtr(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := ParseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
