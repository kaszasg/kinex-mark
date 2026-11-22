package auth

import (
	"errors"
	"io"
	"testing"
)

// errReader fails every read.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("entropy source failed") }

// bytesReader serves a fixed byte sequence, then fails.
type bytesReader struct{ data []byte }

func (r *bytesReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

// withRandReader replaces the package random source for one test. Tests that
// call it must not run in parallel; top-level parallel tests only resume after
// every sequential test has finished, so they never see the replacement.
func withRandReader(t *testing.T, r io.Reader) {
	t.Helper()
	old := randReader
	randReader = r
	t.Cleanup(func() { randReader = old })
}

// testHasher is the cheap hasher the plan prescribes for tests (9.2.1).
func testHasher() *Hasher { return NewHasher(1000, 4) }
