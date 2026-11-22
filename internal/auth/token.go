package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// randReader is the source of every random value in this package. Tests swap
// it for a failing reader to exercise the error paths.
var randReader io.Reader = rand.Reader

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(randReader, b); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return b, nil
}

// TokenBytes is the number of random bytes in a session or challenge token.
const TokenBytes = 32

// NewToken returns a new session or challenge token: 32 random bytes encoded
// as unpadded base64url (43 characters). hash is the SHA-256 of the token
// string; only the hash is stored.
func NewToken() (token string, hash []byte, err error) {
	b, err := randomBytes(TokenBytes)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

// HashToken returns the SHA-256 digest of a presented token (32 bytes).
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// RecoveryCodeAlphabet is the alphabet of recovery codes; it leaves out
// characters that are easy to confuse (i, l, o, 0, 1).
const RecoveryCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// recoveryCodeLength is the number of alphabet characters in one code (about
// 49 bits); the displayed form adds a hyphen after the fifth character.
const recoveryCodeLength = 10

// NewRecoveryCodes returns n distinct recovery codes in the displayed form
// "abcde-23456".
func NewRecoveryCodes(n int) ([]string, error) {
	if n < 0 {
		return nil, fmt.Errorf("generate recovery codes: invalid count %d", n)
	}
	codes := make([]string, 0, n)
	seen := make(map[string]struct{}, n)
	for len(codes) < n {
		s, err := randomString(RecoveryCodeAlphabet, recoveryCodeLength)
		if err != nil {
			return nil, fmt.Errorf("generate recovery codes: %w", err)
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		codes = append(codes, s[:recoveryCodeLength/2]+"-"+s[recoveryCodeLength/2:])
	}
	return codes, nil
}

// NormalizeRecoveryCode lowercases a typed recovery code and removes hyphens
// and whitespace, so "ABCDE-23456", "abcde 23456" and "abcde23456" match.
func NormalizeRecoveryCode(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}
