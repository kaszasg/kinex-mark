package auth

import (
	"bytes"
	"crypto/sha256"
	"regexp"
	"strings"
	"testing"
)

var tokenFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func TestNewToken(t *testing.T) {
	t.Parallel()
	token, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if !tokenFormat.MatchString(token) {
		t.Fatalf("token %q is not 43 base64url characters", token)
	}
	want := sha256.Sum256([]byte(token))
	if !bytes.Equal(hash, want[:]) || len(hash) != 32 {
		t.Fatalf("hash is not the SHA-256 of the token")
	}
	if !bytes.Equal(HashToken(token), hash) {
		t.Fatal("HashToken does not reproduce the returned hash")
	}
	other, otherHash, err := NewToken()
	if err != nil || other == token || bytes.Equal(otherHash, hash) {
		t.Fatalf("second token is not distinct: %v", err)
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	t.Parallel()
	a, b := HashToken("presented-token"), HashToken("presented-token")
	if !bytes.Equal(a, b) || len(a) != 32 {
		t.Fatal("HashToken is not a stable 32-byte digest")
	}
	if bytes.Equal(HashToken("presented-token"), HashToken("presented-tokeN")) {
		t.Fatal("different tokens share a hash")
	}
}

func TestNewTokenRandomFailure(t *testing.T) {
	withRandReader(t, errReader{})
	if _, _, err := NewToken(); err == nil {
		t.Fatal("NewToken succeeded with a failing random source")
	}
}

var recoveryFormat = regexp.MustCompile(`^[abcdefghjkmnpqrstuvwxyz23456789]{5}-[abcdefghjkmnpqrstuvwxyz23456789]{5}$`)

func TestNewRecoveryCodes(t *testing.T) {
	t.Parallel()
	codes, err := NewRecoveryCodes(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 10 {
		t.Fatalf("got %d codes, want 10", len(codes))
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if !recoveryFormat.MatchString(c) {
			t.Fatalf("code %q is not in the xxxxx-xxxxx form", c)
		}
		if seen[c] {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = true
		if n := NormalizeRecoveryCode(c); len(n) != 10 || strings.Contains(n, "-") {
			t.Fatalf("normalized code %q", n)
		}
	}
	if len(RecoveryCodeAlphabet) != 31 || strings.ContainsAny(RecoveryCodeAlphabet, "ilo01") {
		t.Fatalf("alphabet %q is not the planned 31-character set", RecoveryCodeAlphabet)
	}
	none, err := NewRecoveryCodes(0)
	if err != nil || len(none) != 0 {
		t.Fatalf("NewRecoveryCodes(0) = %v, %v", none, err)
	}
}

func TestNewRecoveryCodesSkipsDuplicates(t *testing.T) {
	// rand.Int over 31 values reads one byte per character and keeps its low
	// five bits, so this stream yields "bbbbb-bbbbb" twice and then
	// "ccccc-ccccc".
	stream := append(append(bytes.Repeat([]byte{1}, 10), bytes.Repeat([]byte{1}, 10)...), bytes.Repeat([]byte{2}, 10)...)
	withRandReader(t, &bytesReader{data: stream})
	codes, err := NewRecoveryCodes(2)
	if err != nil {
		t.Fatal(err)
	}
	if codes[0] != "bbbbb-bbbbb" || codes[1] != "ccccc-ccccc" {
		t.Fatalf("codes = %v, want the duplicate skipped", codes)
	}
}

func TestNewRecoveryCodesErrors(t *testing.T) {
	if _, err := NewRecoveryCodes(-1); err == nil {
		t.Fatal("NewRecoveryCodes(-1) succeeded")
	}
	withRandReader(t, errReader{})
	if _, err := NewRecoveryCodes(10); err == nil {
		t.Fatal("NewRecoveryCodes succeeded with a failing random source")
	}
}

func TestNormalizeRecoveryCode(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"abcde-23456":    "abcde23456",
		"ABCDE-23456":    "abcde23456",
		" abcde 23456\t": "abcde23456",
		"ab-cd-e2-34-56": "abcde23456",
		"abcde 23456":    "abcde23456",
		"":               "",
	}
	for in, want := range cases {
		if got := NormalizeRecoveryCode(in); got != want {
			t.Errorf("NormalizeRecoveryCode(%q) = %q, want %q", in, got, want)
		}
	}
}
