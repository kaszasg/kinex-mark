package auth

import (
	"crypto/hmac"
	"crypto/sha1" // RFC 6238 TOTP uses HMAC-SHA1; authenticator apps expect it.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TOTP parameters (RFC 6238, plan 6.5).
const (
	TOTPDigits       = 6
	TOTPPeriod       = 30 // seconds
	TOTPSecretBytes  = 20
	totpSkewSteps    = 1
	totpDigitsModulo = 1_000_000
)

// ErrInvalidTOTPSecret reports a secret that is not valid base32.
var ErrInvalidTOTPSecret = errors.New("invalid TOTP secret")

var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a new 20-byte TOTP secret as upper-case base32
// without padding (32 characters).
func NewTOTPSecret() (string, error) {
	b, err := randomBytes(TOTPSecretBytes)
	if err != nil {
		return "", fmt.Errorf("generate TOTP secret: %w", err)
	}
	return totpEncoding.EncodeToString(b), nil
}

func decodeTOTPSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.TrimRight(strings.ReplaceAll(secret, " ", ""), "="))
	key, err := totpEncoding.DecodeString(s)
	if err != nil || len(key) == 0 {
		return nil, ErrInvalidTOTPSecret
	}
	return key, nil
}

// TOTPStep returns the RFC 6238 time step of t (T0 = 0, 30-second period).
// Times before the Unix epoch map to negative steps.
func TOTPStep(t time.Time) int64 {
	u := t.Unix()
	step := u / TOTPPeriod
	if u < 0 && u%TOTPPeriod != 0 {
		step--
	}
	return step
}

// TOTPCode returns the 6-digit code of secret at time t.
func TOTPCode(secret string, t time.Time) (string, error) {
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return "", err
	}
	step := TOTPStep(t)
	if step < 0 {
		return "", fmt.Errorf("totp code: time %s is before the Unix epoch", t.UTC().Format(time.RFC3339))
	}
	return hotp(key, uint64(step)), nil
}

// hotp computes the RFC 4226 HOTP value truncated to TOTPDigits digits.
func hotp(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", TOTPDigits, value%totpDigitsModulo)
}

// VerifyTOTP checks a 6-digit code against the steps around now (current step
// plus or minus one). The candidate steps are tried in ascending order and
// only a step greater than lastStep is accepted, which prevents replay. On
// success it returns the matched step, which the caller stores as the new
// last step.
func VerifyTOTP(secret, code string, now time.Time, lastStep int64) (step int64, ok bool) {
	if len(code) != TOTPDigits || !isDigits(code) {
		return 0, false
	}
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return 0, false
	}
	current := TOTPStep(now)
	for s := current - totpSkewSteps; s <= current+totpSkewSteps; s++ {
		if s < 0 || s <= lastStep {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(hotp(key, uint64(s))), []byte(code)) == 1 {
			return s, true
		}
	}
	return 0, false
}

func isDigits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// TOTPURI returns the otpauth:// provisioning URI that authenticator apps
// read from the QR code, for example
// otpauth://totp/Kinex%20Mark:alice?secret=...&issuer=Kinex%20Mark&algorithm=SHA1&digits=6&period=30.
// Issuer and account are percent-encoded (a space becomes %20).
func TOTPURI(issuer, account, secret string) string {
	var b strings.Builder
	b.WriteString("otpauth://totp/")
	b.WriteString(percentEncode(issuer))
	b.WriteByte(':')
	b.WriteString(percentEncode(account))
	b.WriteString("?secret=")
	b.WriteString(percentEncode(secret))
	b.WriteString("&issuer=")
	b.WriteString(percentEncode(issuer))
	fmt.Fprintf(&b, "&algorithm=SHA1&digits=%d&period=%d", TOTPDigits, TOTPPeriod)
	return b.String()
}

// percentEncode escapes every byte outside the RFC 3986 unreserved set.
func percentEncode(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0f])
	}
	return b.String()
}
