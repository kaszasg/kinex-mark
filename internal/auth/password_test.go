package auth

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

var hashFormat = regexp.MustCompile(`^pbkdf2-sha256\$1000\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)

func TestHashVerifyRoundTrip(t *testing.T) {
	t.Parallel()
	h := testHasher()
	encoded, err := h.Hash("correct horse battery")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !hashFormat.MatchString(encoded) {
		t.Fatalf("hash %q does not match the stored format", encoded)
	}
	ok, rehash, err := h.Verify(encoded, "correct horse battery")
	if err != nil || !ok || rehash {
		t.Fatalf("Verify(correct) = %v, %v, %v; want true, false, nil", ok, rehash, err)
	}
	ok, rehash, err = h.Verify(encoded, "correct horse batterY")
	if err != nil || ok || rehash {
		t.Fatalf("Verify(wrong) = %v, %v, %v; want false, false, nil", ok, rehash, err)
	}
}

func TestHashUsesUniqueSalt(t *testing.T) {
	t.Parallel()
	h := testHasher()
	a, err := h.Hash("same password 123")
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Hash("same password 123")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("two hashes of the same password are equal")
	}
	if strings.Split(a, "$")[2] == strings.Split(b, "$")[2] {
		t.Fatal("two hashes share a salt")
	}
}

func TestVerifyNeedsRehash(t *testing.T) {
	t.Parallel()
	old := NewHasher(500, 1)
	encoded, err := old.Hash("rehash me please")
	if err != nil {
		t.Fatal(err)
	}
	cur := testHasher()
	ok, rehash, err := cur.Verify(encoded, "rehash me please")
	if err != nil || !ok || !rehash {
		t.Fatalf("lower iterations: Verify = %v, %v, %v; want true, true, nil", ok, rehash, err)
	}
	ok, rehash, err = cur.Verify(encoded, "wrong password!!")
	if err != nil || ok || rehash {
		t.Fatalf("lower iterations, wrong password: Verify = %v, %v, %v; want false, false, nil", ok, rehash, err)
	}
	ok, rehash, err = NewHasher(200, 1).Verify(encoded, "rehash me please")
	if err != nil || !ok || rehash {
		t.Fatalf("higher stored iterations: Verify = %v, %v, %v; want true, false, nil", ok, rehash, err)
	}
}

func TestVerifyMalformedHash(t *testing.T) {
	t.Parallel()
	h := testHasher()
	cases := map[string]string{
		"empty":             "",
		"other scheme":      "bcrypt$1000$c2FsdHNhbHRzYWx0c2FsdA$a2V5",
		"too few parts":     "pbkdf2-sha256$1000$c2FsdA",
		"too many parts":    "pbkdf2-sha256$1000$c2FsdA$a2V5$extra",
		"iterations text":   "pbkdf2-sha256$many$c2FsdA$a2V5",
		"iterations zero":   "pbkdf2-sha256$0$c2FsdA$a2V5",
		"iterations neg":    "pbkdf2-sha256$-5$c2FsdA$a2V5",
		"iterations huge":   "pbkdf2-sha256$100000001$c2FsdA$a2V5",
		"salt not base64":   "pbkdf2-sha256$1000$!!!$a2V5",
		"salt empty":        "pbkdf2-sha256$1000$$a2V5",
		"salt padded":       "pbkdf2-sha256$1000$c2FsdA==$a2V5",
		"key not base64":    "pbkdf2-sha256$1000$c2FsdA$***",
		"key empty":         "pbkdf2-sha256$1000$c2FsdA$",
		"upper-case scheme": "PBKDF2-SHA256$1000$c2FsdA$a2V5",
	}
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ok, rehash, err := h.Verify(encoded, "whatever password")
			if !errors.Is(err, ErrMalformedHash) || ok || rehash {
				t.Fatalf("Verify(%q) = %v, %v, %v; want ErrMalformedHash", encoded, ok, rehash, err)
			}
		})
	}
}

func TestDefaultIterationsEncoding(t *testing.T) {
	t.Parallel()
	if DefaultIterations != 600000 {
		t.Fatalf("DefaultIterations = %d, want 600000", DefaultIterations)
	}
	h := NewHasher(DefaultIterations, 1)
	if h.Iterations() != 600000 {
		t.Fatalf("Iterations() = %d", h.Iterations())
	}
	encoded, err := h.Hash("production cost check")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "pbkdf2-sha256$600000$") {
		t.Fatalf("hash %q does not encode 600000 iterations", encoded)
	}
}

func TestNewHasherClampsArguments(t *testing.T) {
	t.Parallel()
	h := NewHasher(0, -3)
	if h.Iterations() != 1 || cap(h.sem) != 1 {
		t.Fatalf("NewHasher(0, -3): iterations %d, concurrency %d; want 1, 1", h.Iterations(), cap(h.sem))
	}
	encoded, err := h.Hash("tiny cost works")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _, err := h.Verify(encoded, "tiny cost works"); !ok || err != nil {
		t.Fatalf("Verify = %v, %v", ok, err)
	}
	if got := cap(testHasher().sem); got != 4 {
		t.Fatalf("concurrency limit = %d, want 4", got)
	}
}

func TestHasherConcurrentUse(t *testing.T) {
	t.Parallel()
	h := NewHasher(1000, 2)
	encoded, err := h.Hash("shared password 1")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan string, 16)
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				if ok, _, err := h.Verify(encoded, "shared password 1"); !ok || err != nil {
					errs <- "verify failed"
				}
				return
			}
			h.DummyVerify("anything")
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if len(h.sem) != 0 {
		t.Fatalf("semaphore holds %d slots after all calls returned", len(h.sem))
	}
}

func TestDummyVerifyNeverMatches(t *testing.T) {
	t.Parallel()
	h := testHasher()
	if !hashFormat.MatchString(h.dummy) {
		t.Fatalf("dummy hash %q is not in the stored format", h.dummy)
	}
	h.DummyVerify("")
	if ok, _, err := h.Verify(h.dummy, ""); ok || err != nil {
		t.Fatalf("dummy hash matched: %v, %v", ok, err)
	}
}

func TestHashRandomFailure(t *testing.T) {
	withRandReader(t, errReader{})
	if _, err := testHasher().Hash("some password 1"); err == nil || !strings.Contains(err.Error(), "generate salt") {
		t.Fatalf("Hash error = %v, want a salt generation error", err)
	}
}

func TestCheckPasswordPolicy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, password, username, current, want string
	}{
		{"11 runes", "abcdefghijk", "", "", CodePasswordTooShort},
		{"12 runes", "abcdefghijkl", "", "", ""},
		{"11 accented runes", "árvíztűrőtü", "", "", CodePasswordTooShort},
		{"12 accented runes", "árvíztűrőtük", "", "", ""},
		{"empty", "", "alice", "", CodePasswordTooShort},
		{"128 runes", strings.Repeat("abcdefgh", 16), "", "", ""},
		{"129 runes", strings.Repeat("abcdefgh", 16) + "x", "", "", CodePasswordTooLong},
		{"128 accented runes", strings.Repeat("áéíóúöőü", 16), "", "", ""},
		{"129 accented runes", strings.Repeat("áéíóúöőü", 16) + "ű", "", "", CodePasswordTooLong},
		{"contains username", "my-alice-secret-1", "alice", "", CodePasswordContainsUsername},
		{"contains username other case", "my-ALICE-secret-1", "Alice", "", CodePasswordContainsUsername},
		{"contains accented username", "XÁRPÁD-jelszó-2026", "árpád", "", CodePasswordContainsUsername},
		{"short username ignored", "ab-cdefghijk-1", "ab", "", ""},
		{"3-rune username checked", "xyzabc-defgh1", "abc", "", CodePasswordContainsUsername},
		{"5 distinct runes", "abcdeabcdeabcde", "", "", CodePasswordTooSimple},
		{"6 distinct runes", "abcdefabcdefabcdef", "", "", ""},
		{"5 distinct accented runes", "áéíóúáéíóúáéíóú", "", "", CodePasswordTooSimple},
		{"reused", "a long enough pass", "", "a long enough pass", CodePasswordReused},
		{"differs from current", "a long enough pass", "", "a long enough pas", ""},
		{"no current skips reuse", "a long enough pass", "", "", ""},
		{"order: short before username", "alice1", "alice", "", CodePasswordTooShort},
		{"order: username before simple", "alicealicealice", "alice", "", CodePasswordContainsUsername},
		{"order: simple before reused", "aaaaaaaaaaaaaaa", "", "aaaaaaaaaaaaaaa", CodePasswordTooSimple},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := CheckPasswordPolicy(c.password, c.username, c.current); got != c.want {
				t.Fatalf("CheckPasswordPolicy(%q, %q, %q) = %q, want %q", c.password, c.username, c.current, got, c.want)
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	t.Parallel()
	p, err := GeneratePassword(20)
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(p) != 20 {
		t.Fatalf("length = %d, want 20", len(p))
	}
	for _, r := range p {
		if !strings.ContainsRune(PasswordAlphabet, r) {
			t.Fatalf("rune %q is outside the alphabet", r)
		}
	}
	if code := CheckPasswordPolicy(p, "admin", ""); code != "" {
		t.Fatalf("generated password fails the policy: %s", code)
	}
	if strings.ContainsAny(PasswordAlphabet, "IOlo01") || len(PasswordAlphabet) != 56 {
		t.Fatalf("alphabet %q is not the planned 56-character set", PasswordAlphabet)
	}
	q, err := GeneratePassword(20)
	if err != nil {
		t.Fatal(err)
	}
	if p == q {
		t.Fatal("two generated passwords are equal")
	}
}

func TestGeneratePasswordDistinctRunes(t *testing.T) {
	t.Parallel()
	// With 6 characters about a quarter of the draws repeat a character, so
	// this loop exercises the regeneration path many times.
	for range 300 {
		p, err := GeneratePassword(6)
		if err != nil {
			t.Fatal(err)
		}
		if distinctRunes(p) != 6 {
			t.Fatalf("password %q has fewer than 6 distinct runes", p)
		}
	}
	short, err := GeneratePassword(3)
	if err != nil || len(short) != 3 {
		t.Fatalf("GeneratePassword(3) = %q, %v", short, err)
	}
}

func TestGeneratePasswordErrors(t *testing.T) {
	for _, n := range []int{0, -1} {
		if _, err := GeneratePassword(n); err == nil {
			t.Fatalf("GeneratePassword(%d) succeeded", n)
		}
	}
	withRandReader(t, errReader{})
	if _, err := GeneratePassword(20); err == nil {
		t.Fatal("GeneratePassword succeeded with a failing random source")
	}
}
