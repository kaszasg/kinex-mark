package auth

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

// rfcSecret is the RFC 6238 SHA-1 test key "12345678901234567890" in base32.
var rfcSecret = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))

func TestTOTPCodeRFC6238Vectors(t *testing.T) {
	t.Parallel()
	if rfcSecret != "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" {
		t.Fatalf("unexpected base32 test secret %q", rfcSecret)
	}
	// RFC 6238 appendix B, SHA-1 column, truncated to 6 digits.
	vectors := []struct {
		unix int64
		code string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	for _, v := range vectors {
		got, err := TOTPCode(rfcSecret, time.Unix(v.unix, 0))
		if err != nil {
			t.Fatalf("TOTPCode(T=%d): %v", v.unix, err)
		}
		if got != v.code {
			t.Errorf("TOTPCode(T=%d) = %s, want %s", v.unix, got, v.code)
		}
	}
}

func TestTOTPSecretFormsAccepted(t *testing.T) {
	t.Parallel()
	at := time.Unix(59, 0)
	forms := []string{
		strings.ToLower(rfcSecret),
		"GEZD GNBV GY3T QOJQ GEZD GNBV GY3T QOJQ",
		rfcSecret + "====",
	}
	for _, s := range forms {
		got, err := TOTPCode(s, at)
		if err != nil || got != "287082" {
			t.Errorf("TOTPCode(%q) = %q, %v; want 287082", s, got, err)
		}
	}
}

func TestTOTPCodeErrors(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "not base32!", "1"} {
		if _, err := TOTPCode(s, time.Unix(59, 0)); err != ErrInvalidTOTPSecret {
			t.Errorf("TOTPCode(%q) error = %v, want ErrInvalidTOTPSecret", s, err)
		}
	}
	if _, err := TOTPCode(rfcSecret, time.Unix(-1, 0)); err == nil {
		t.Error("TOTPCode before the epoch succeeded")
	}
}

func TestTOTPStep(t *testing.T) {
	t.Parallel()
	cases := map[int64]int64{0: 0, 29: 0, 30: 1, 59: 1, -1: -1, -30: -1, -31: -2}
	for unix, want := range cases {
		if got := TOTPStep(time.Unix(unix, 0)); got != want {
			t.Errorf("TOTPStep(%d) = %d, want %d", unix, got, want)
		}
	}
}

func TestNewTOTPSecret(t *testing.T) {
	t.Parallel()
	s, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 32 || s != strings.ToUpper(s) || strings.Contains(s, "=") {
		t.Fatalf("secret %q is not 32 upper-case base32 characters without padding", s)
	}
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil || len(raw) != 20 {
		t.Fatalf("secret decodes to %d bytes (%v), want 20", len(raw), err)
	}
	other, err := NewTOTPSecret()
	if err != nil || other == s {
		t.Fatalf("second secret %q, %v", other, err)
	}
	if _, err := TOTPCode(s, time.Now()); err != nil {
		t.Fatalf("a new secret is not usable: %v", err)
	}
}

func TestNewTOTPSecretRandomFailure(t *testing.T) {
	withRandReader(t, errReader{})
	if _, err := NewTOTPSecret(); err == nil {
		t.Fatal("NewTOTPSecret succeeded with a failing random source")
	}
}

func codeAt(t *testing.T, step int64) string {
	t.Helper()
	c, err := TOTPCode(rfcSecret, time.Unix(step*TOTPPeriod, 0))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestVerifyTOTPWindow(t *testing.T) {
	t.Parallel()
	now := time.Unix(1234567890, 0)
	cur := TOTPStep(now)
	cases := []struct {
		name     string
		step     int64
		lastStep int64
		wantOK   bool
	}{
		{"current step", cur, 0, true},
		{"one step behind", cur - 1, 0, true},
		{"one step ahead", cur + 1, 0, true},
		{"two steps behind", cur - 2, 0, false},
		{"two steps ahead", cur + 2, 0, false},
		{"replayed current step", cur, cur, false},
		{"step below last step", cur - 1, cur, false},
		{"next step after last", cur + 1, cur, true},
		{"current after previous", cur, cur - 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			step, ok := VerifyTOTP(rfcSecret, codeAt(t, c.step), now, c.lastStep)
			if ok != c.wantOK {
				t.Fatalf("VerifyTOTP ok = %v, want %v", ok, c.wantOK)
			}
			if ok && step != c.step {
				t.Fatalf("VerifyTOTP step = %d, want %d", step, c.step)
			}
			if !ok && step != 0 {
				t.Fatalf("rejected code returned step %d", step)
			}
		})
	}
}

func TestVerifyTOTPRejectsBadInput(t *testing.T) {
	t.Parallel()
	now := time.Unix(59, 0)
	good := "287082"
	if _, ok := VerifyTOTP(rfcSecret, good, now, 0); !ok {
		t.Fatal("valid code rejected")
	}
	for _, code := range []string{"", "28708", "2870820", "28708a", " 87082", "２87082"} {
		if _, ok := VerifyTOTP(rfcSecret, code, now, 0); ok {
			t.Errorf("code %q accepted", code)
		}
	}
	if _, ok := VerifyTOTP("not base32!", good, now, 0); ok {
		t.Error("invalid secret accepted")
	}
}

func TestVerifyTOTPNearEpoch(t *testing.T) {
	t.Parallel()
	// At T = 0 the window holds steps -1, 0 and 1; step -1 is skipped.
	now := time.Unix(0, 0)
	step, ok := VerifyTOTP(rfcSecret, codeAt(t, 0), now, -5)
	if !ok || step != 0 {
		t.Fatalf("VerifyTOTP at the epoch = %d, %v; want 0, true", step, ok)
	}
	step, ok = VerifyTOTP(rfcSecret, codeAt(t, 1), now, -5)
	if !ok || step != 1 {
		t.Fatalf("VerifyTOTP next step at the epoch = %d, %v; want 1, true", step, ok)
	}
}

func TestTOTPURI(t *testing.T) {
	t.Parallel()
	got := TOTPURI("Kinex Mark", "alice", "JBSWY3DPEHPK3PXP")
	want := "otpauth://totp/Kinex%20Mark:alice?secret=JBSWY3DPEHPK3PXP&issuer=Kinex%20Mark&algorithm=SHA1&digits=6&period=30"
	if got != want {
		t.Fatalf("TOTPURI =\n%s\nwant\n%s", got, want)
	}
	got = TOTPURI("Kinex Mark", "a b&c:d/é", "S")
	want = "otpauth://totp/Kinex%20Mark:a%20b%26c%3Ad%2F%C3%A9?secret=S&issuer=Kinex%20Mark&algorithm=SHA1&digits=6&period=30"
	if got != want {
		t.Fatalf("TOTPURI with special characters =\n%s\nwant\n%s", got, want)
	}
	if got := TOTPURI("x", "j.doe_2-x~", "S"); !strings.Contains(got, ":j.doe_2-x~?") {
		t.Fatalf("unreserved characters were escaped: %s", got)
	}
}
