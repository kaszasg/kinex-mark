package config

import (
	"encoding/base64"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net/netip"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// load runs Load with a fresh flag set and a map-backed environment.
func load(t *testing.T, env map[string]string, args ...string) (Config, error) {
	t.Helper()
	fs := flag.NewFlagSet("kinex-mark", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return Load(fs, args, func(k string) string { return env[k] })
}

func mustLoad(t *testing.T, env map[string]string, args ...string) Config {
	t.Helper()
	cfg, err := load(t, env, args...)
	if err != nil {
		t.Fatalf("Load(%v, %v): unexpected error: %v", env, args, err)
	}
	return cfg
}

func validKey() string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", SecretKeyBytes)))
}

func TestDefaults(t *testing.T) {
	t.Parallel()
	cfg := mustLoad(t, nil)
	want := Config{
		Addr:           DefaultAddr,
		DataDir:        DefaultDataDir,
		SecureCookies:  SecureCookiesAuto,
		TrustedProxies: []netip.Prefix{},
		LogLevel:       slog.LevelInfo,
		LogFormat:      "text",
		FetchWorkers:   2,
		FetchTimeout:   10 * time.Second,
		AdminUser:      "admin",
		SecretKeyFile:  filepath.Join(DefaultDataDir, "secret.key"),
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("defaults:\n got %+v\nwant %+v", cfg, want)
	}
	if got := cfg.EnvLockedSettings(); got == nil || len(got) != 0 {
		t.Fatalf("EnvLockedSettings() = %#v, want empty non-nil slice", got)
	}
}

func TestEnvironmentValues(t *testing.T) {
	t.Parallel()
	env := map[string]string{
		EnvAddr:               "127.0.0.1:9000",
		EnvDataDir:            "/var/lib/kinex",
		EnvBaseURL:            "https://Mark.Example.com/",
		EnvSecureCookies:      "TRUE",
		EnvTrustedProxy:       " 10.0.0.1 , 192.168.0.0/16,::ffff:172.16.0.5, 2001:db8::/32 ",
		EnvAllowPrivateFetch:  "false",
		EnvAllowLoopbackFetch: "true",
		EnvTLSCert:            "cert.pem",
		EnvTLSKey:             "key.pem",
		EnvLogLevel:           "DEBUG",
		EnvLogFormat:          "json",
		EnvFetchWorkers:       "16",
		EnvFetchTimeout:       "1m",
		EnvAdminUser:          "Root.Admin_1-x",
		EnvAdminPassword:      "a long password",
		EnvSecretKey:          validKey(),
		EnvSecretKeyFile:      "/run/secrets/key",
	}
	cfg := mustLoad(t, env)
	if cfg.Addr != "127.0.0.1:9000" || cfg.DataDir != "/var/lib/kinex" {
		t.Errorf("addr/data = %q/%q", cfg.Addr, cfg.DataDir)
	}
	if cfg.BaseURL == nil || cfg.BaseURL.String() != "https://mark.example.com" {
		t.Errorf("BaseURL = %v, want https://mark.example.com", cfg.BaseURL)
	}
	if cfg.SecureCookies != SecureCookiesTrue {
		t.Errorf("SecureCookies = %q", cfg.SecureCookies)
	}
	wantProxies := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.1/32"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("172.16.0.5/32"),
		netip.MustParsePrefix("2001:db8::/32"),
	}
	if !reflect.DeepEqual(cfg.TrustedProxies, wantProxies) {
		t.Errorf("TrustedProxies = %v, want %v", cfg.TrustedProxies, wantProxies)
	}
	if cfg.AllowPrivateFetch == nil || *cfg.AllowPrivateFetch {
		t.Errorf("AllowPrivateFetch = %v, want pointer to false", cfg.AllowPrivateFetch)
	}
	if !cfg.AllowLoopbackFetch {
		t.Error("AllowLoopbackFetch = false, want true")
	}
	if cfg.TLSCertFile != "cert.pem" || cfg.TLSKeyFile != "key.pem" {
		t.Errorf("TLS = %q/%q", cfg.TLSCertFile, cfg.TLSKeyFile)
	}
	if cfg.LogLevel != slog.LevelDebug || cfg.LogFormat != "json" {
		t.Errorf("log = %v/%q", cfg.LogLevel, cfg.LogFormat)
	}
	if cfg.FetchWorkers != 16 || cfg.FetchTimeout != time.Minute {
		t.Errorf("fetch = %d/%v", cfg.FetchWorkers, cfg.FetchTimeout)
	}
	if cfg.AdminUser != "Root.Admin_1-x" || cfg.AdminPassword != "a long password" {
		t.Errorf("admin = %q/%q", cfg.AdminUser, cfg.AdminPassword)
	}
	if cfg.SecretKey != validKey() || cfg.SecretKeyFile != "/run/secrets/key" {
		t.Errorf("secret = %q/%q", cfg.SecretKey, cfg.SecretKeyFile)
	}
	if got := cfg.EnvLockedSettings(); !reflect.DeepEqual(got, []string{"allowPrivateFetch"}) {
		t.Errorf("EnvLockedSettings() = %v", got)
	}
}

func TestFlagOverridesEnvironment(t *testing.T) {
	t.Parallel()
	env := map[string]string{
		EnvAddr:              ":7000",
		EnvDataDir:           "env-data",
		EnvFetchWorkers:      "3",
		EnvLogLevel:          "error",
		EnvAllowPrivateFetch: "false",
		EnvAdminUser:         "envadmin",
	}
	cfg := mustLoad(t, env,
		"-addr", ":9000", "-data", "flag-data", "-fetch-workers", "4", "-log-level", "warn",
		"-allow-private-fetch", "-allow-loopback-fetch", "-admin-user", "flagadmin",
		"-fetch-timeout", "2s", "-secure-cookies", "false", "-log-format", "JSON",
		"-trusted-proxy", "127.0.0.1", "-base-url", "http://localhost:8080",
		"-tls-cert", "c.pem", "-tls-key", "k.pem", "-secret-key-file", "k.key",
	)
	if cfg.Addr != ":9000" || cfg.DataDir != "flag-data" || cfg.FetchWorkers != 4 {
		t.Errorf("flags did not win: %+v", cfg)
	}
	if cfg.LogLevel != slog.LevelWarn || cfg.LogFormat != "json" || cfg.SecureCookies != SecureCookiesFalse {
		t.Errorf("log/cookies = %v/%q/%q", cfg.LogLevel, cfg.LogFormat, cfg.SecureCookies)
	}
	if cfg.AllowPrivateFetch == nil || !*cfg.AllowPrivateFetch || !cfg.AllowLoopbackFetch {
		t.Errorf("bare boolean flags not true: %v %v", cfg.AllowPrivateFetch, cfg.AllowLoopbackFetch)
	}
	if cfg.AdminUser != "flagadmin" || cfg.FetchTimeout != 2*time.Second {
		t.Errorf("admin/timeout = %q/%v", cfg.AdminUser, cfg.FetchTimeout)
	}
	if cfg.BaseURL.String() != "http://localhost:8080" || cfg.SecretKeyFile != "k.key" {
		t.Errorf("base/key file = %v/%q", cfg.BaseURL, cfg.SecretKeyFile)
	}
	if len(cfg.TrustedProxies) != 1 || cfg.TrustedProxies[0] != netip.MustParsePrefix("127.0.0.1/32") {
		t.Errorf("TrustedProxies = %v", cfg.TrustedProxies)
	}
}

func TestSecretKeyFileFollowsDataDir(t *testing.T) {
	t.Parallel()
	cfg := mustLoad(t, map[string]string{EnvDataDir: "/data"})
	if want := filepath.Join("/data", "secret.key"); cfg.SecretKeyFile != want {
		t.Fatalf("SecretKeyFile = %q, want %q", cfg.SecretKeyFile, want)
	}
}

func TestAllowLoopbackFetch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  string
		args []string
		want bool
	}{
		{"default", "", nil, false},
		{"env true", "true", nil, true},
		{"env false", "false", nil, false},
		{"env upper case", "TRUE", nil, true},
		{"bare flag", "", []string{"-allow-loopback-fetch"}, true},
		{"flag false overrides env", "true", []string{"-allow-loopback-fetch=false"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := mustLoad(t, map[string]string{EnvAllowLoopbackFetch: tc.env}, tc.args...)
			if cfg.AllowLoopbackFetch != tc.want {
				t.Fatalf("AllowLoopbackFetch = %v, want %v", cfg.AllowLoopbackFetch, tc.want)
			}
			if cfg.AllowPrivateFetch != nil {
				t.Fatalf("AllowPrivateFetch = %v, want nil", *cfg.AllowPrivateFetch)
			}
			if len(cfg.EnvLockedSettings()) != 0 {
				t.Fatalf("loopback must not lock a system setting: %v", cfg.EnvLockedSettings())
			}
		})
	}
}

func TestAllowPrivateFetchTriState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		env  string
		want *bool
	}{
		{"", nil},
		{"true", ptr(true)},
		{"false", ptr(false)},
	} {
		cfg := mustLoad(t, map[string]string{EnvAllowPrivateFetch: tc.env})
		if !reflect.DeepEqual(cfg.AllowPrivateFetch, tc.want) {
			t.Errorf("env %q: AllowPrivateFetch = %v, want %v", tc.env, cfg.AllowPrivateFetch, tc.want)
		}
		locked := len(cfg.EnvLockedSettings()) == 1
		if locked != (tc.want != nil) {
			t.Errorf("env %q: EnvLockedSettings() = %v", tc.env, cfg.EnvLockedSettings())
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{"addr without port", map[string]string{EnvAddr: "localhost"}, nil, "config: APP_ADDR: must be host:port or :port"},
		{"addr bad port", map[string]string{EnvAddr: ":http"}, nil, "config: APP_ADDR: port must be a number between 0 and 65535"},
		{"addr port too big", nil, []string{"-addr", ":70000"}, "config: -addr: port must be a number between 0 and 65535"},
		{"data empty flag", nil, []string{"-data", " "}, "config: -data: must not be empty"},
		{"base url with path", map[string]string{EnvBaseURL: "https://mark.example.com/app"}, nil, "config: APP_BASE_URL: path must be empty or /"},
		{"base url ftp", map[string]string{EnvBaseURL: "ftp://mark.example.com"}, nil, "config: APP_BASE_URL: scheme must be http or https"},
		{"base url relative", map[string]string{EnvBaseURL: "mark.example.com"}, nil, "config: APP_BASE_URL: scheme must be http or https"},
		{"base url unparsable", map[string]string{EnvBaseURL: "https://exa mple.com"}, nil, "config: APP_BASE_URL: must be an absolute http or https URL"},
		{"base url no host", map[string]string{EnvBaseURL: "https:///path"}, nil, "config: APP_BASE_URL: must be an absolute http or https URL with a host"},
		{"base url opaque", map[string]string{EnvBaseURL: "https:mark.example.com"}, nil, "config: APP_BASE_URL: must be an absolute http or https URL with a host"},
		{"base url credentials", map[string]string{EnvBaseURL: "https://u:p@mark.example.com"}, nil, "config: APP_BASE_URL: must not contain credentials"},
		{"base url query", map[string]string{EnvBaseURL: "https://mark.example.com/?a=1"}, nil, "config: APP_BASE_URL: must not contain a query"},
		{"base url empty query", map[string]string{EnvBaseURL: "https://mark.example.com/?"}, nil, "config: APP_BASE_URL: must not contain a query"},
		{"base url fragment", map[string]string{EnvBaseURL: "https://mark.example.com/#top"}, nil, "config: APP_BASE_URL: must not contain a fragment"},
		{"base url empty fragment", map[string]string{EnvBaseURL: "https://mark.example.com#"}, nil, "config: APP_BASE_URL: must not contain a fragment"},
		{"base url port zero", map[string]string{EnvBaseURL: "https://mark.example.com:0"}, nil, "config: APP_BASE_URL: port must be a number between 1 and 65535"},
		{"base url empty port", map[string]string{EnvBaseURL: "https://mark.example.com:"}, nil, "config: APP_BASE_URL: port must be a number between 1 and 65535"},
		{"secure cookies unknown", map[string]string{EnvSecureCookies: "yes"}, nil, "config: APP_SECURE_COOKIES: must be auto, true or false"},
		{"bad CIDR", map[string]string{EnvTrustedProxy: "10.0.0.0/33"}, nil, `config: APP_TRUSTED_PROXY: invalid IP address or CIDR "10.0.0.0/33"`},
		{"bad proxy IP", map[string]string{EnvTrustedProxy: "10.0.0.1,proxy.local"}, nil, `config: APP_TRUSTED_PROXY: invalid IP address or CIDR "proxy.local"`},
		{"proxy with zone", map[string]string{EnvTrustedProxy: "fe80::1%eth0"}, nil, `config: APP_TRUSTED_PROXY: invalid IP address or CIDR "fe80::1%eth0"`},
		{"allow private invalid", map[string]string{EnvAllowPrivateFetch: "1"}, nil, "config: APP_ALLOW_PRIVATE_FETCH: must be true or false"},
		{"allow loopback invalid", map[string]string{EnvAllowLoopbackFetch: "on"}, nil, "config: APP_ALLOW_LOOPBACK_FETCH: must be true or false"},
		{"cert without key", map[string]string{EnvTLSCert: "c.pem"}, nil, "config: APP_TLS_CERT: requires APP_TLS_KEY"},
		{"key without cert", nil, []string{"-tls-key", "k.pem"}, "config: -tls-key: requires APP_TLS_CERT"},
		{"log level unknown", map[string]string{EnvLogLevel: "trace"}, nil, "config: APP_LOG_LEVEL: must be debug, info, warn or error"},
		{"log format unknown", map[string]string{EnvLogFormat: "xml"}, nil, "config: APP_LOG_FORMAT: must be text or json"},
		{"workers 0", map[string]string{EnvFetchWorkers: "0"}, nil, "config: APP_FETCH_WORKERS: must be between 1 and 16"},
		{"workers 17", map[string]string{EnvFetchWorkers: "17"}, nil, "config: APP_FETCH_WORKERS: must be between 1 and 16"},
		{"workers flag 0", nil, []string{"-fetch-workers", "0"}, "config: -fetch-workers: must be between 1 and 16"},
		{"workers not a number", map[string]string{EnvFetchWorkers: "two"}, nil, "config: APP_FETCH_WORKERS: must be an integer between 1 and 16"},
		{"timeout 500ms", map[string]string{EnvFetchTimeout: "500ms"}, nil, "config: APP_FETCH_TIMEOUT: must be a duration between 1s and 1m0s"},
		{"timeout 61s", map[string]string{EnvFetchTimeout: "61s"}, nil, "config: APP_FETCH_TIMEOUT: must be a duration between 1s and 1m0s"},
		{"timeout not a duration", map[string]string{EnvFetchTimeout: "10"}, nil, "config: APP_FETCH_TIMEOUT: must be a duration between 1s and 1m0s"},
		{"admin user with space", map[string]string{EnvAdminUser: "the admin"}, nil, "config: APP_ADMIN_USER: invalid username"},
		{"admin user with slash", map[string]string{EnvAdminUser: "ad/min"}, nil, "config: APP_ADMIN_USER: invalid username"},
		{"admin user 2 characters", map[string]string{EnvAdminUser: "ad"}, nil, "config: APP_ADMIN_USER: invalid username"},
		{"admin user 65 characters", map[string]string{EnvAdminUser: strings.Repeat("a", 65)}, nil, "config: APP_ADMIN_USER: invalid username"},
		{"admin user non-ASCII", map[string]string{EnvAdminUser: "admin" + string(rune(0xe1))}, nil, "config: APP_ADMIN_USER: invalid username"},
		{"admin user flag", nil, []string{"-admin-user", "a b"}, "config: -admin-user: invalid username"},
		{"secret key not base64", map[string]string{EnvSecretKey: "not base64!"}, nil, "config: APP_SECRET_KEY: must be 32 bytes encoded in base64"},
		{"secret key 16 bytes", map[string]string{EnvSecretKey: base64.StdEncoding.EncodeToString(make([]byte, 16))}, nil, "config: APP_SECRET_KEY: must be 32 bytes encoded in base64"},
		{"positional argument", nil, []string{"serve"}, `config: unexpected argument "serve"`},
		{"bare private flag then value", nil, []string{"-allow-private-fetch", "false"}, `config: unexpected argument "false"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := load(t, tc.env, tc.args...)
			if err == nil {
				t.Fatalf("Load: expected error %q, got nil", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("Load error:\n got %q\nwant %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSecretKeyNeverInError(t *testing.T) {
	t.Parallel()
	secret := "c2VjcmV0LXZhbHVlLXRoYXQtaXMtdG9vLXNob3J0"
	_, err := load(t, map[string]string{EnvSecretKey: secret})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("error must exist and hide the key: %v", err)
	}
}

func TestSecretKeyEncodings(t *testing.T) {
	t.Parallel()
	raw := []byte(strings.Repeat("\xfb", SecretKeyBytes))
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		key := enc.EncodeToString(raw)
		if _, err := load(t, map[string]string{EnvSecretKey: key + "\n"}); err != nil {
			t.Errorf("key %q rejected: %v", key, err)
		}
	}
}

func TestUsernameBoundaries(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"abc", strings.Repeat("z", 64), "a.b", "A_B-c", "123"} {
		if cfg := mustLoad(t, map[string]string{EnvAdminUser: name}); cfg.AdminUser != name {
			t.Errorf("AdminUser = %q, want %q", cfg.AdminUser, name)
		}
	}
}

func TestBaseURLForms(t *testing.T) {
	t.Parallel()
	for raw, want := range map[string]string{
		"https://mark.example.com":       "https://mark.example.com",
		"https://mark.example.com/":      "https://mark.example.com",
		"HTTP://LOCALHOST:5173":          "http://localhost:5173",
		"http://[::1]:8080/":             "http://[::1]:8080",
		" https://mark.example.com:8443": "https://mark.example.com:8443",
	} {
		cfg := mustLoad(t, map[string]string{EnvBaseURL: raw})
		if cfg.BaseURL.String() != want {
			t.Errorf("%q: BaseURL = %q, want %q", raw, cfg.BaseURL.String(), want)
		}
		if cfg.BaseURL.Path != "" || cfg.BaseURL.RawQuery != "" || cfg.BaseURL.Fragment != "" {
			t.Errorf("%q: BaseURL keeps path, query or fragment: %#v", raw, cfg.BaseURL)
		}
	}
}

func TestTrustedProxyForms(t *testing.T) {
	t.Parallel()
	cfg := mustLoad(t, map[string]string{EnvTrustedProxy: "10.1.2.3/8,::1,::ffff:10.0.0.0/104,,"})
	want := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}
	if !reflect.DeepEqual(cfg.TrustedProxies, want) {
		t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
}

func TestLogLevels(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]slog.Level{
		"debug": slog.LevelDebug, "info": slog.LevelInfo, "Warn": slog.LevelWarn, "error": slog.LevelError,
	} {
		if cfg := mustLoad(t, map[string]string{EnvLogLevel: in}); cfg.LogLevel != want {
			t.Errorf("%q: LogLevel = %v, want %v", in, cfg.LogLevel, want)
		}
	}
}

func TestAdminCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		want    *AdminCommand
		wantErr string
	}{
		{"none", nil, nil, ""},
		{"unlock", []string{"-unlock-user", "alice"}, &AdminCommand{Kind: AdminCommandUnlockUser, Username: "alice"}, ""},
		{"reset", []string{"-data", "d", "-reset-password", "admin"}, &AdminCommand{Kind: AdminCommandResetPassword, Username: "admin"}, ""},
		{"both", []string{"-unlock-user", "alice", "-reset-password", "alice"}, nil, "config: -unlock-user and -reset-password cannot be combined"},
		{"unlock with healthcheck", []string{"-unlock-user", "alice", "-healthcheck"}, nil, "config: -unlock-user cannot be combined with -healthcheck"},
		{"reset with version", []string{"-version", "-reset-password", "alice"}, nil, "config: -reset-password cannot be combined with -version"},
		{"unlock invalid name", []string{"-unlock-user", "al ice"}, nil, "config: -unlock-user: invalid username"},
		{"reset empty name", []string{"-reset-password="}, nil, "config: -reset-password: invalid username"},
		{"reset short name", []string{"-reset-password", "ab"}, nil, "config: -reset-password: invalid username"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := load(t, nil, tc.args...)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(cfg.AdminCommand, tc.want) {
				t.Fatalf("AdminCommand = %+v, want %+v", cfg.AdminCommand, tc.want)
			}
		})
	}
}

func TestHealthCheckAndVersion(t *testing.T) {
	t.Parallel()
	cfg := mustLoad(t, nil, "-healthcheck", "-version")
	if !cfg.HealthCheck || !cfg.ShowVersion || cfg.AdminCommand != nil {
		t.Fatalf("HealthCheck/ShowVersion = %v/%v, AdminCommand = %v", cfg.HealthCheck, cfg.ShowVersion, cfg.AdminCommand)
	}
}

func TestFlagErrors(t *testing.T) {
	t.Parallel()
	if _, err := load(t, nil, "-h"); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("-h: error = %v, want flag.ErrHelp", err)
	}
	_, err := load(t, nil, "-no-such-flag")
	if err == nil || !strings.HasPrefix(err.Error(), "config: flag provided but not defined") {
		t.Fatalf("unknown flag: error = %v", err)
	}
	_, err = load(t, nil, "-healthcheck=maybe")
	if err == nil || !strings.HasPrefix(err.Error(), "config: ") {
		t.Fatalf("bad bool flag: error = %v", err)
	}
}

func TestHelpTextIsEnglishAndComplete(t *testing.T) {
	t.Parallel()
	fs := flag.NewFlagSet("kinex-mark", flag.ContinueOnError)
	var out strings.Builder
	fs.SetOutput(&out)
	if _, err := Load(fs, []string{"-h"}, func(string) string { return "" }); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v", err)
	}
	for _, name := range []string{
		"addr", "data", "base-url", "secure-cookies", "trusted-proxy", "allow-private-fetch",
		"allow-loopback-fetch", "tls-cert", "tls-key", "log-level", "log-format", "fetch-workers",
		"fetch-timeout", "admin-user", "secret-key-file", "healthcheck", "version", "unlock-user",
		"reset-password",
	} {
		if !strings.Contains(out.String(), "  -"+name) {
			t.Errorf("help text misses -%s", name)
		}
	}
	for _, secret := range []string{"admin-password", "secret-key "} {
		if strings.Contains(out.String(), "-"+secret) {
			t.Errorf("help text must not offer a -%s flag", secret)
		}
	}
	if !strings.Contains(out.String(), "listen address") {
		t.Error("help text misses the listen address usage")
	}
}

func TestTextValueZero(t *testing.T) {
	t.Parallel()
	if got := (textValue{}).String(); got != "" {
		t.Fatalf("zero textValue String() = %q", got)
	}
}
