package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// KeyBytes is the size of the server key.
const KeyBytes = 32

const (
	sealPrefix    = "v1:"
	gcmNonceBytes = 12
	gcmTagBytes   = 16
	keyFileMode   = 0o600
	keyDirMode    = 0o750
	keyCheckLabel = "key-check"
	keyCheckData  = "kinex-mark"
)

// ErrOpenSealed reports a sealed value that cannot be opened: wrong format,
// wrong key or wrong associated data.
var ErrOpenSealed = errors.New("open sealed value: authentication failed")

// Keyring holds the 32-byte server key. It seals TOTP secrets with
// AES-256-GCM and derives purpose-bound HMAC-SHA256 values (recovery code
// hashes, throttle keys, the key check value).
type Keyring struct {
	key  [KeyBytes]byte
	aead cipher.AEAD
}

// NewKeyring returns a Keyring for a raw 32-byte key.
func NewKeyring(key []byte) (*Keyring, error) {
	if len(key) != KeyBytes {
		return nil, fmt.Errorf("secret key: must be %d bytes, got %d", KeyBytes, len(key))
	}
	k := &Keyring{}
	copy(k.key[:], key)
	block, err := aes.NewCipher(k.key[:])
	if err != nil {
		return nil, fmt.Errorf("secret key: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secret key: %w", err)
	}
	k.aead = aead
	return k, nil
}

// LoadOrCreateKeyring returns the server keyring. A non-empty envKey
// (APP_SECRET_KEY, base64) takes precedence and the file is not touched.
// Otherwise the key is read from filePath; when the file does not exist, a new
// random key is written there (base64, mode 0600) and created is true.
// Errors never contain key material.
func LoadOrCreateKeyring(envKey, filePath string) (kr *Keyring, created bool, err error) {
	if strings.TrimSpace(envKey) != "" {
		kr, err = decodeKey(envKey)
		if err != nil {
			return nil, false, fmt.Errorf("APP_SECRET_KEY: %w", err)
		}
		return kr, false, nil
	}
	if filePath == "" {
		return nil, false, errors.New("secret key: no APP_SECRET_KEY and no key file path")
	}
	data, err := os.ReadFile(filePath)
	switch {
	case err == nil:
		kr, err = decodeKey(string(data))
		if err != nil {
			return nil, false, fmt.Errorf("secret key file %s: %w", filePath, err)
		}
		return kr, false, nil
	case errors.Is(err, fs.ErrNotExist):
		kr, err = createKeyFile(filePath)
		if err != nil {
			return nil, false, err
		}
		return kr, true, nil
	default:
		return nil, false, fmt.Errorf("read secret key file: %w", err)
	}
}

func decodeKey(s string) (*Keyring, error) {
	s = strings.TrimSpace(s)
	var raw []byte
	var err error
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		raw, err = enc.DecodeString(s)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, errors.New("invalid base64")
	}
	return NewKeyring(raw)
}

func createKeyFile(filePath string) (*Keyring, error) {
	key, err := randomBytes(KeyBytes)
	if err != nil {
		return nil, fmt.Errorf("generate secret key: %w", err)
	}
	kr, err := NewKeyring(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), keyDirMode); err != nil {
		return nil, fmt.Errorf("create secret key directory: %w", err)
	}
	// O_EXCL: never overwrite a key that appeared in the meantime.
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, keyFileMode)
	if err != nil {
		return nil, fmt.Errorf("create secret key file: %w", err)
	}
	_, werr := f.WriteString(base64.StdEncoding.EncodeToString(key) + "\n")
	serr := f.Sync()
	cerr := f.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("write secret key file: %w", err)
	}
	return kr, nil
}

// Seal encrypts plaintext with AES-256-GCM under a fresh 12-byte random nonce,
// binding it to associated (for example "totp:42"). The result is
// "v1:" + base64(nonce || ciphertext).
func (k *Keyring) Seal(plaintext []byte, associated string) (string, error) {
	nonce, err := randomBytes(gcmNonceBytes)
	if err != nil {
		return "", fmt.Errorf("seal: %w", err)
	}
	out := k.aead.Seal(nonce, nonce, plaintext, []byte(associated))
	return sealPrefix + base64.StdEncoding.EncodeToString(out), nil
}

// Open reverses Seal. It fails with ErrOpenSealed when the value is
// malformed, was sealed with another key or with other associated data.
func (k *Keyring) Open(sealed, associated string) ([]byte, error) {
	rest, found := strings.CutPrefix(sealed, sealPrefix)
	if !found {
		return nil, ErrOpenSealed
	}
	raw, err := base64.StdEncoding.DecodeString(rest)
	if err != nil || len(raw) < gcmNonceBytes+gcmTagBytes {
		return nil, ErrOpenSealed
	}
	plain, err := k.aead.Open(nil, raw[:gcmNonceBytes], raw[gcmNonceBytes:], []byte(associated))
	if err != nil {
		return nil, ErrOpenSealed
	}
	return plain, nil
}

// MAC returns hex(HMAC-SHA256(HMAC-SHA256(key, purpose), data)): a 64-character
// value bound to both the server key and the purpose.
func (k *Keyring) MAC(purpose string, data []byte) string {
	sub := hmac.New(sha256.New, k.key[:])
	sub.Write([]byte(purpose))
	mac := hmac.New(sha256.New, sub.Sum(nil))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// CheckValue returns MAC("key-check", "kinex-mark"). The store keeps it, so a
// start with a different key is detected before any secret is misread.
func (k *Keyring) CheckValue() string {
	return k.MAC(keyCheckLabel, []byte(keyCheckData))
}
