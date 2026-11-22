package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testKey(fill byte) []byte { return bytes.Repeat([]byte{fill}, KeyBytes) }

func mustKeyring(t *testing.T, fill byte) *Keyring {
	t.Helper()
	kr, err := NewKeyring(testKey(fill))
	if err != nil {
		t.Fatal(err)
	}
	return kr
}

func TestLoadOrCreateKeyringCreatesAndLoadsFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "secret.key")
	kr, created, err := LoadOrCreateKeyring("", path)
	if err != nil || !created {
		t.Fatalf("first load: created %v, err %v; want true, nil", created, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("key file %q does not end with a newline", data)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(raw) != KeyBytes {
		t.Fatalf("key file holds %d bytes (%v), want 32 in base64", len(raw), err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("key file mode = %o, want 600", perm)
		}
	}
	again, created, err := LoadOrCreateKeyring("", path)
	if err != nil || created {
		t.Fatalf("second load: created %v, err %v; want false, nil", created, err)
	}
	if again.CheckValue() != kr.CheckValue() {
		t.Fatal("reloaded key differs from the created key")
	}
	fromRaw := mustKeyringFrom(t, raw)
	if fromRaw.CheckValue() != kr.CheckValue() {
		t.Fatal("file content is not the key in use")
	}
}

func mustKeyringFrom(t *testing.T, raw []byte) *Keyring {
	t.Helper()
	kr, err := NewKeyring(raw)
	if err != nil {
		t.Fatal(err)
	}
	return kr
}

func TestLoadOrCreateKeyringEnvKeyWins(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.key")
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(testKey(2))), 0o600); err != nil {
		t.Fatal(err)
	}
	want := mustKeyring(t, 1).CheckValue()
	env := base64.StdEncoding.EncodeToString(testKey(1))
	kr, created, err := LoadOrCreateKeyring(env, path)
	if err != nil || created {
		t.Fatalf("env key: created %v, err %v", created, err)
	}
	if kr.CheckValue() != want {
		t.Fatal("the file key was used although APP_SECRET_KEY is set")
	}
	missing := filepath.Join(dir, "absent.key")
	if _, _, err := LoadOrCreateKeyring(env, missing); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a key file was created although APP_SECRET_KEY is set: %v", err)
	}
	if _, _, err := LoadOrCreateKeyring(env, ""); err != nil {
		t.Fatalf("env key without a file path: %v", err)
	}
}

func TestLoadOrCreateKeyringEncodings(t *testing.T) {
	t.Parallel()
	key := testKey(0xfb) // 0xfb produces '+' and '/' (or '-' and '_') in base64
	want := mustKeyringFrom(t, key).CheckValue()
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		s := "  " + enc.EncodeToString(key) + "\r\n"
		kr, _, err := LoadOrCreateKeyring(s, "")
		if err != nil {
			t.Fatalf("LoadOrCreateKeyring(%q): %v", s, err)
		}
		if kr.CheckValue() != want {
			t.Fatalf("encoding %q decoded to another key", s)
		}
	}
}

func TestLoadOrCreateKeyringErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	shortKey := base64.StdEncoding.EncodeToString(testKey(3)[:16])
	notBase64 := "this is not base64 at all!"

	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	blocker := write("blocker", "a regular file")
	// Windows reports a missing path below a regular file as "not found", so
	// the creation step fails; POSIX systems already fail the read (ENOTDIR).
	blockedWant := "create secret key directory"
	if runtime.GOOS != "windows" {
		blockedWant = "read secret key file"
	}

	cases := []struct {
		name, env, path, want string
	}{
		{"env invalid base64", notBase64, "", "APP_SECRET_KEY: invalid base64"},
		{"env wrong length", shortKey, "", "APP_SECRET_KEY: secret key: must be 32 bytes, got 16"},
		{"no source", "", "", "no APP_SECRET_KEY and no key file path"},
		{"blank env and no path", "   ", "", "no APP_SECRET_KEY and no key file path"},
		{"file invalid base64", "", write("bad.key", notBase64), "invalid base64"},
		{"file wrong length", "", write("short.key", shortKey), "must be 32 bytes, got 16"},
		{"empty file", "", write("empty.key", ""), "must be 32 bytes, got 0"},
		{"path is a directory", "", dir, "read secret key file"},
		{"parent is a file", "", filepath.Join(blocker, "secret.key"), blockedWant},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kr, created, err := LoadOrCreateKeyring(c.env, c.path)
			if err == nil || kr != nil || created {
				t.Fatalf("got %v, %v, %v; want an error", kr, created, err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err, c.want)
			}
			if env := strings.TrimSpace(c.env); env != "" && strings.Contains(err.Error(), env) {
				t.Fatalf("error %q leaks the key material", err)
			}
		})
	}
}

func TestLoadOrCreateKeyringRandomFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	withRandReader(t, errReader{})
	if _, _, err := LoadOrCreateKeyring("", path); err == nil || !strings.Contains(err.Error(), "generate secret key") {
		t.Fatalf("error = %v, want a key generation error", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a key file was left behind: %v", err)
	}
}

func TestNewKeyringRejectsWrongLength(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 16, 31, 33, 64} {
		if _, err := NewKeyring(make([]byte, n)); err == nil {
			t.Errorf("NewKeyring with %d bytes succeeded", n)
		}
	}
}

func TestNewKeyringCopiesKey(t *testing.T) {
	t.Parallel()
	key := testKey(4)
	kr := mustKeyringFrom(t, key)
	before := kr.CheckValue()
	key[0] ^= 0xff
	if kr.CheckValue() != before {
		t.Fatal("the keyring aliases the caller's slice")
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	t.Parallel()
	kr := mustKeyring(t, 5)
	secret := []byte("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	sealed, err := kr.Seal(secret, "totp:42")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sealed, "v1:") {
		t.Fatalf("sealed value %q lacks the v1: prefix", sealed)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sealed, "v1:"))
	if err != nil || len(raw) != 12+len(secret)+16 {
		t.Fatalf("sealed payload is %d bytes (%v), want nonce, ciphertext and tag", len(raw), err)
	}
	if bytes.Contains(raw, secret) {
		t.Fatal("sealed payload contains the plaintext")
	}
	plain, err := kr.Open(sealed, "totp:42")
	if err != nil || !bytes.Equal(plain, secret) {
		t.Fatalf("Open = %q, %v", plain, err)
	}
	again, err := kr.Seal(secret, "totp:42")
	if err != nil || again == sealed {
		t.Fatalf("two seals are identical; the nonce is not random (%v)", err)
	}
	empty, err := kr.Seal(nil, "a")
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := kr.Open(empty, "a"); err != nil || len(plain) != 0 {
		t.Fatalf("Open(empty) = %q, %v", plain, err)
	}
}

func TestOpenRejects(t *testing.T) {
	t.Parallel()
	kr := mustKeyring(t, 6)
	sealed, err := kr.Seal([]byte("secret"), "totp:1")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(sealed, "v1:"))
	tampered := bytes.Clone(raw)
	tampered[len(tampered)-1] ^= 0x01
	cases := map[string]struct {
		kr         *Keyring
		sealed, ad string
	}{
		"other associated data": {kr, sealed, "totp:2"},
		"other key":             {mustKeyring(t, 7), sealed, "totp:1"},
		"missing prefix":        {kr, strings.TrimPrefix(sealed, "v1:"), "totp:1"},
		"other version":         {kr, "v2:" + strings.TrimPrefix(sealed, "v1:"), "totp:1"},
		"not base64":            {kr, "v1:***", "totp:1"},
		"too short":             {kr, "v1:" + base64.StdEncoding.EncodeToString(raw[:27]), "totp:1"},
		"tampered":              {kr, "v1:" + base64.StdEncoding.EncodeToString(tampered), "totp:1"},
		"empty":                 {kr, "", "totp:1"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plain, err := c.kr.Open(c.sealed, c.ad)
			if !errors.Is(err, ErrOpenSealed) || plain != nil {
				t.Fatalf("Open = %q, %v; want ErrOpenSealed", plain, err)
			}
		})
	}
}

func TestSealRandomFailure(t *testing.T) {
	kr := mustKeyring(t, 8)
	withRandReader(t, errReader{})
	if _, err := kr.Seal([]byte("x"), "totp:1"); err == nil {
		t.Fatal("Seal succeeded with a failing random source")
	}
}

func referenceMAC(key []byte, purpose string, data []byte) string {
	sub := hmac.New(sha256.New, key)
	sub.Write([]byte(purpose))
	mac := hmac.New(sha256.New, sub.Sum(nil))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestMAC(t *testing.T) {
	t.Parallel()
	kr := mustKeyring(t, 9)
	a := kr.MAC("recovery-code", []byte("abcde23456"))
	if a != referenceMAC(testKey(9), "recovery-code", []byte("abcde23456")) {
		t.Fatal("MAC is not hex(HMAC(HMAC(key, purpose), data))")
	}
	if len(a) != 64 || strings.ToLower(a) != a {
		t.Fatalf("MAC %q is not 64 lower-case hex characters", a)
	}
	if kr.MAC("recovery-code", []byte("abcde23456")) != a {
		t.Fatal("MAC is not deterministic")
	}
	if kr.MAC("throttle-username", []byte("abcde23456")) == a {
		t.Fatal("MAC does not depend on the purpose")
	}
	if kr.MAC("recovery-code", []byte("abcde23457")) == a {
		t.Fatal("MAC does not depend on the data")
	}
	if mustKeyring(t, 10).MAC("recovery-code", []byte("abcde23456")) == a {
		t.Fatal("MAC does not depend on the key")
	}
	// The purpose and the data are not simply concatenated.
	if kr.MAC("ab", []byte("c")) == kr.MAC("a", []byte("bc")) {
		t.Fatal("MAC collides across the purpose boundary")
	}
}

func TestCheckValue(t *testing.T) {
	t.Parallel()
	kr := mustKeyring(t, 11)
	want := referenceMAC(testKey(11), "key-check", []byte("kinex-mark"))
	if kr.CheckValue() != want || kr.CheckValue() != kr.MAC("key-check", []byte("kinex-mark")) {
		t.Fatal("CheckValue is not MAC(\"key-check\", \"kinex-mark\")")
	}
	if mustKeyring(t, 11).CheckValue() != want {
		t.Fatal("CheckValue is not stable for the same key")
	}
	if mustKeyring(t, 12).CheckValue() == want {
		t.Fatal("CheckValue does not distinguish keys")
	}
}
