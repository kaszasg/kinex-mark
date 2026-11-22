// Package auth holds the pure security primitives of Kinex Mark: password
// hashing and policy, TOTP, session and challenge tokens, recovery codes, the
// server keyring and the login throttle arithmetic. It performs no database
// access; the store applies the throttle functions inside its own transaction.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// DefaultIterations is the production PBKDF2-HMAC-SHA256 iteration count.
const DefaultIterations = 600_000

const (
	hashScheme    = "pbkdf2-sha256"
	saltLength    = 16
	keyLength     = 32
	maxIterations = 100_000_000 // guards Verify against absurd stored parameters
)

// Password policy limits (plan 6.4).
const (
	MinPasswordRunes      = 12
	MaxPasswordRunes      = 128
	MinDistinctRunes      = 6
	minUsernameRunesCheck = 3
)

// Field codes returned by CheckPasswordPolicy (plan 5.3, 6.4).
const (
	CodePasswordTooShort         = "password_too_short"
	CodePasswordTooLong          = "password_too_long"
	CodePasswordContainsUsername = "password_contains_username"
	CodePasswordTooSimple        = "password_too_simple"
	CodePasswordReused           = "password_reused"
)

// PasswordAlphabet is the alphabet of generated passwords; it leaves out
// characters that are easy to confuse (I, O, l, o, 0, 1).
const PasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// ErrMalformedHash reports a stored password hash that cannot be parsed.
var ErrMalformedHash = errors.New("malformed password hash")

// Hasher hashes and verifies passwords with PBKDF2-HMAC-SHA256. A semaphore
// bounds the number of concurrent derivations, so a burst of logins cannot
// exhaust the CPU.
type Hasher struct {
	iterations int
	sem        chan struct{}
	dummy      string
}

// NewHasher returns a Hasher. Production uses NewHasher(DefaultIterations, 4);
// tests use NewHasher(1000, 4). Values below 1 are raised to 1.
func NewHasher(iterations, maxConcurrent int) *Hasher {
	if iterations < 1 {
		iterations = 1
	}
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	// The dummy hash carries a random salt and a random key, so verifying
	// against it costs exactly one full derivation and never succeeds. No
	// derivation is needed to build it, which keeps start-up fast.
	salt := make([]byte, saltLength)
	key := make([]byte, keyLength)
	_, _ = rand.Read(salt) // crypto/rand.Read never returns an error since Go 1.24
	_, _ = rand.Read(key)
	return &Hasher{
		iterations: iterations,
		sem:        make(chan struct{}, maxConcurrent),
		dummy:      encodeHash(iterations, salt, key),
	}
}

// Iterations returns the iteration count used for new hashes.
func (h *Hasher) Iterations() int { return h.iterations }

func (h *Hasher) acquire() func() {
	h.sem <- struct{}{}
	return func() { <-h.sem }
}

// Hash derives a new hash with a fresh 16-byte salt and returns it in the
// stored form pbkdf2-sha256$<iterations>$<salt>$<key> (raw standard base64).
func (h *Hasher) Hash(password string) (string, error) {
	salt, err := randomBytes(saltLength)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	release := h.acquire()
	defer release()
	key, err := pbkdf2.Key(sha256.New, password, salt, h.iterations, keyLength)
	if err != nil {
		return "", fmt.Errorf("derive password key: %w", err)
	}
	return encodeHash(h.iterations, salt, key), nil
}

// Verify checks password against an encoded hash. The parameters come from
// the encoded value; needsRehash is true when the password matched and the
// stored iteration count is below the Hasher's current count.
func (h *Hasher) Verify(encoded, password string) (ok, needsRehash bool, err error) {
	iterations, salt, want, err := decodeHash(encoded)
	if err != nil {
		return false, false, err
	}
	release := h.acquire()
	defer release()
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false, false, fmt.Errorf("derive password key: %w", err)
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, false, nil
	}
	return true, iterations < h.iterations, nil
}

// DummyVerify spends the same time as a real verification. Login calls it for
// an unknown username, so the response time does not reveal whether an
// account exists.
func (h *Hasher) DummyVerify(password string) {
	_, _, _ = h.Verify(h.dummy, password)
}

func encodeHash(iterations int, salt, key []byte) string {
	return hashScheme + "$" + strconv.Itoa(iterations) + "$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key)
}

func decodeHash(encoded string) (iterations int, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != hashScheme {
		return 0, nil, nil, ErrMalformedHash
	}
	iterations, err = strconv.Atoi(parts[1])
	if err != nil || iterations < 1 || iterations > maxIterations {
		return 0, nil, nil, ErrMalformedHash
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) == 0 {
		return 0, nil, nil, ErrMalformedHash
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(key) == 0 {
		return 0, nil, nil, ErrMalformedHash
	}
	return iterations, salt, key, nil
}

// CheckPasswordPolicy returns "" when password satisfies the policy of plan
// 6.4, otherwise the field code of the first rule it breaks. The rules run in
// order: length (12..128 runes), username containment (case-insensitive, only
// when the username has at least 3 runes), at least 6 distinct runes, and,
// when current is not empty (password change), difference from current.
func CheckPasswordPolicy(password, username, current string) string {
	n := utf8.RuneCountInString(password)
	if n < MinPasswordRunes {
		return CodePasswordTooShort
	}
	if n > MaxPasswordRunes {
		return CodePasswordTooLong
	}
	if utf8.RuneCountInString(username) >= minUsernameRunesCheck &&
		strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
		return CodePasswordContainsUsername
	}
	if distinctRunes(password) < MinDistinctRunes {
		return CodePasswordTooSimple
	}
	if current != "" && password == current {
		return CodePasswordReused
	}
	return ""
}

func distinctRunes(s string) int {
	seen := make(map[rune]struct{}, len(s))
	for _, r := range s {
		seen[r] = struct{}{}
	}
	return len(seen)
}

// GeneratePassword returns a uniformly random password of the given length
// over PasswordAlphabet. Passwords of at least MinDistinctRunes characters are
// regenerated until they hold that many distinct characters, so a generated
// password of policy length always passes CheckPasswordPolicy (apart from the
// username rule, which the caller cannot hit with this alphabet in practice).
func GeneratePassword(length int) (string, error) {
	if length < 1 {
		return "", fmt.Errorf("generate password: invalid length %d", length)
	}
	for {
		s, err := randomString(PasswordAlphabet, length)
		if err != nil {
			return "", fmt.Errorf("generate password: %w", err)
		}
		if length < MinDistinctRunes || distinctRunes(s) >= MinDistinctRunes {
			return s, nil
		}
	}
}

// randomString draws length characters uniformly from the ASCII alphabet.
func randomString(alphabet string, length int) (string, error) {
	limit := big.NewInt(int64(len(alphabet)))
	var b strings.Builder
	b.Grow(length)
	for range length {
		i, err := rand.Int(randReader, limit)
		if err != nil {
			return "", err
		}
		b.WriteByte(alphabet[i.Int64()])
	}
	return b.String(), nil
}
