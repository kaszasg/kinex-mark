// Package config parses command-line flags and environment variables into a
// validated Config (plan 3.3.2, 3.4). Precedence: flag, environment variable,
// default. The package depends on the standard library only.
package config

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SecureCookies is the cookie security mode: "auto", "true" or "false" (6.2).
type SecureCookies string

// Cookie security modes.
const (
	SecureCookiesAuto  SecureCookies = "auto"
	SecureCookiesTrue  SecureCookies = "true"
	SecureCookiesFalse SecureCookies = "false"
)

// Admin command kinds (6.12).
const (
	AdminCommandUnlockUser    = "unlock-user"
	AdminCommandResetPassword = "reset-password"
)

// AdminCommand is an offline recovery command; nil in Config means a normal start.
type AdminCommand struct {
	Kind     string // AdminCommandUnlockUser or AdminCommandResetPassword
	Username string
}

// Config is the validated runtime configuration.
type Config struct {
	Addr               string
	DataDir            string
	BaseURL            *url.URL // nil when not set; scheme and host only (path, query and fragment are empty)
	SecureCookies      SecureCookies
	TrustedProxies     []netip.Prefix
	AllowPrivateFetch  *bool // nil: the system setting decides
	AllowLoopbackFetch bool  // APP_ALLOW_LOOPBACK_FETCH only; no system setting (6.8)
	TLSCertFile        string
	TLSKeyFile         string
	LogLevel           slog.Level
	LogFormat          string // "text" or "json"
	FetchWorkers       int
	FetchTimeout       time.Duration
	AdminUser          string
	AdminPassword      string // environment only
	SecretKey          string // environment only (base64)
	SecretKeyFile      string // defaults to <DataDir>/secret.key
	HealthCheck        bool
	ShowVersion        bool
	AdminCommand       *AdminCommand // nil: normal start
}

// Defaults (3.4).
const (
	DefaultAddr          = ":8080"
	DefaultDataDir       = "./data"
	DefaultSecureCookies = SecureCookiesAuto
	DefaultLogLevel      = "info"
	DefaultLogFormat     = "text"
	DefaultFetchWorkers  = 2
	DefaultFetchTimeout  = 10 * time.Second
	DefaultAdminUser     = "admin"
	SecretKeyFileName    = "secret.key"
)

// Validation limits (3.4).
const (
	MinFetchWorkers = 1
	MaxFetchWorkers = 16
	MinFetchTimeout = time.Second
	MaxFetchTimeout = 60 * time.Second
	SecretKeyBytes  = 32
)

// Environment variable names (3.4).
const (
	EnvAddr               = "APP_ADDR"
	EnvDataDir            = "APP_DATA_DIR"
	EnvBaseURL            = "APP_BASE_URL"
	EnvSecureCookies      = "APP_SECURE_COOKIES"
	EnvTrustedProxy       = "APP_TRUSTED_PROXY"
	EnvAllowPrivateFetch  = "APP_ALLOW_PRIVATE_FETCH"
	EnvAllowLoopbackFetch = "APP_ALLOW_LOOPBACK_FETCH"
	EnvTLSCert            = "APP_TLS_CERT"
	EnvTLSKey             = "APP_TLS_KEY"
	EnvLogLevel           = "APP_LOG_LEVEL"
	EnvLogFormat          = "APP_LOG_FORMAT"
	EnvFetchWorkers       = "APP_FETCH_WORKERS"
	EnvFetchTimeout       = "APP_FETCH_TIMEOUT"
	EnvAdminUser          = "APP_ADMIN_USER"
	EnvAdminPassword      = "APP_ADMIN_PASSWORD"
	EnvSecretKey          = "APP_SECRET_KEY"
	EnvSecretKeyFile      = "APP_SECRET_KEY_FILE"
)

// settingAllowPrivateFetch is the system setting key locked by
// APP_ALLOW_PRIVATE_FETCH (model.SettingAllowPrivateFetch; config imports
// the standard library only).
const settingAllowPrivateFetch = "allowPrivateFetch"

// Username rule of 5.12, repeated here because config imports no internal package.
const (
	usernameMinLen = 3
	usernameMaxLen = 64
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// validUsername reports whether s satisfies the 5.12 username rule.
func validUsername(s string) bool {
	return len(s) >= usernameMinLen && len(s) <= usernameMaxLen && usernamePattern.MatchString(s)
}

// textValue is a string flag. With isBool set it also accepts the bare form
// (-allow-loopback-fetch means -allow-loopback-fetch=true).
type textValue struct {
	s      *string
	isBool bool
}

func (v textValue) String() string {
	if v.s == nil {
		return ""
	}
	return *v.s
}

func (v textValue) Set(s string) error { *v.s = s; return nil }

func (v textValue) IsBoolFlag() bool { return v.isBool }

// option is one configurable value: its flag name (empty when there is no
// flag), its environment variable (empty when there is none) and the raw text.
type option struct {
	flag string
	env  string
	raw  string
}

// label names the source of the value in error messages: the flag when it
// was given on the command line, otherwise the environment variable.
func (o *option) label(set map[string]bool) string {
	if o.flag != "" && (set[o.flag] || o.env == "") {
		return "-" + o.flag
	}
	return o.env
}

// Load parses args with fs and reads the environment through getenv. It
// validates every value and returns the first error in the form
// "config: APP_FETCH_WORKERS: must be between 1 and 16". Flag parse errors
// are returned wrapped, so errors.Is(err, flag.ErrHelp) still works.
func Load(fs *flag.FlagSet, args []string, getenv func(string) string) (Config, error) {
	env := func(key, def string) string {
		if key == "" {
			return def
		}
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}
	opt := func(flagName, envName, def string) *option {
		return &option{flag: flagName, env: envName, raw: env(envName, def)}
	}

	var (
		addr          = opt("addr", EnvAddr, DefaultAddr)
		dataDir       = opt("data", EnvDataDir, DefaultDataDir)
		baseURL       = opt("base-url", EnvBaseURL, "")
		secureCookies = opt("secure-cookies", EnvSecureCookies, string(DefaultSecureCookies))
		trustedProxy  = opt("trusted-proxy", EnvTrustedProxy, "")
		allowPrivate  = opt("allow-private-fetch", EnvAllowPrivateFetch, "")
		allowLoopback = opt("allow-loopback-fetch", EnvAllowLoopbackFetch, "false")
		tlsCert       = opt("tls-cert", EnvTLSCert, "")
		tlsKey        = opt("tls-key", EnvTLSKey, "")
		logLevel      = opt("log-level", EnvLogLevel, DefaultLogLevel)
		logFormat     = opt("log-format", EnvLogFormat, DefaultLogFormat)
		fetchWorkers  = opt("fetch-workers", EnvFetchWorkers, strconv.Itoa(DefaultFetchWorkers))
		fetchTimeout  = opt("fetch-timeout", EnvFetchTimeout, DefaultFetchTimeout.String())
		adminUser     = opt("admin-user", EnvAdminUser, DefaultAdminUser)
		secretKey     = opt("", EnvSecretKey, "")
		secretKeyFile = opt("secret-key-file", EnvSecretKeyFile, "")
		unlockUser    = opt("unlock-user", "", "")
		resetPassword = opt("reset-password", "", "")
	)

	var cfg Config
	str := func(o *option, usage string) { fs.Var(textValue{s: &o.raw}, o.flag, usage) }
	boolStr := func(o *option, usage string) { fs.Var(textValue{s: &o.raw, isBool: true}, o.flag, usage) }
	str(addr, "listen address")
	str(dataDir, "data directory (app.db, secret.key)")
	str(baseURL, "external URL of the user interface, for example https://mark.example.com (required behind a reverse proxy)")
	str(secureCookies, "secure cookie mode: auto, true or false")
	str(trustedProxy, "comma-separated IP addresses or CIDR ranges of trusted reverse proxies")
	boolStr(allowPrivate, "allow metadata fetches from private networks: true or false; overrides and locks the system setting")
	boolStr(allowLoopback, "allow metadata fetches from loopback and link-local addresses (test environments only)")
	str(tlsCert, "TLS certificate file (PEM); requires -tls-key")
	str(tlsKey, "TLS private key file (PEM); requires -tls-cert")
	str(logLevel, "log level: debug, info, warn or error")
	str(logFormat, "log format: text or json")
	str(fetchWorkers, "number of metadata fetch workers (1-16)")
	str(fetchTimeout, "timeout of one metadata fetch, a Go duration between 1s and 60s")
	str(adminUser, "username of the first administrator (empty database only)")
	str(secretKeyFile, "server key file, created when missing (default <data>/secret.key)")
	fs.BoolVar(&cfg.HealthCheck, "healthcheck", false, "check the running server, then exit")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print the version, then exit")
	str(unlockUser, "offline recovery: clear the login throttle of the named user, then exit")
	str(resetPassword, "offline recovery: generate a new password for the named user, then exit")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if rest := fs.Args(); len(rest) > 0 {
		return Config{}, fmt.Errorf("config: unexpected argument %q", rest[0])
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	fail := func(o *option, format string, a ...any) error {
		return fmt.Errorf("config: %s: %s", o.label(set), fmt.Sprintf(format, a...))
	}

	// Listen address.
	if err := checkAddr(addr.raw); err != nil {
		return Config{}, fail(addr, "%s", err)
	}
	cfg.Addr = addr.raw

	// Data directory.
	if strings.TrimSpace(dataDir.raw) == "" {
		return Config{}, fail(dataDir, "must not be empty")
	}
	cfg.DataDir = dataDir.raw

	// External base URL.
	if baseURL.raw != "" {
		u, err := parseBaseURL(baseURL.raw)
		if err != nil {
			return Config{}, fail(baseURL, "%s", err)
		}
		cfg.BaseURL = u
	}

	// Secure cookie mode.
	switch mode := SecureCookies(strings.ToLower(strings.TrimSpace(secureCookies.raw))); mode {
	case SecureCookiesAuto, SecureCookiesTrue, SecureCookiesFalse:
		cfg.SecureCookies = mode
	default:
		return Config{}, fail(secureCookies, "must be auto, true or false")
	}

	// Trusted proxies.
	proxies, err := parseProxies(trustedProxy.raw)
	if err != nil {
		return Config{}, fail(trustedProxy, "%s", err)
	}
	cfg.TrustedProxies = proxies

	// Fetch policy switches.
	if strings.TrimSpace(allowPrivate.raw) != "" {
		v, err := parseBool(allowPrivate.raw)
		if err != nil {
			return Config{}, fail(allowPrivate, "%s", err)
		}
		cfg.AllowPrivateFetch = &v
	}
	if strings.TrimSpace(allowLoopback.raw) != "" {
		v, err := parseBool(allowLoopback.raw)
		if err != nil {
			return Config{}, fail(allowLoopback, "%s", err)
		}
		cfg.AllowLoopbackFetch = v
	}

	// TLS: both files or neither.
	cfg.TLSCertFile, cfg.TLSKeyFile = tlsCert.raw, tlsKey.raw
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile == "" {
		return Config{}, fail(tlsCert, "requires %s", tlsKey.label(set))
	}
	if cfg.TLSKeyFile != "" && cfg.TLSCertFile == "" {
		return Config{}, fail(tlsKey, "requires %s", tlsCert.label(set))
	}

	// Logging.
	level, err := parseLogLevel(logLevel.raw)
	if err != nil {
		return Config{}, fail(logLevel, "%s", err)
	}
	cfg.LogLevel = level
	switch format := strings.ToLower(strings.TrimSpace(logFormat.raw)); format {
	case "text", "json":
		cfg.LogFormat = format
	default:
		return Config{}, fail(logFormat, "must be text or json")
	}

	// Metadata fetcher.
	workers, err := strconv.Atoi(strings.TrimSpace(fetchWorkers.raw))
	if err != nil {
		return Config{}, fail(fetchWorkers, "must be an integer between %d and %d", MinFetchWorkers, MaxFetchWorkers)
	}
	if workers < MinFetchWorkers || workers > MaxFetchWorkers {
		return Config{}, fail(fetchWorkers, "must be between %d and %d", MinFetchWorkers, MaxFetchWorkers)
	}
	cfg.FetchWorkers = workers
	timeout, err := time.ParseDuration(strings.TrimSpace(fetchTimeout.raw))
	if err != nil || timeout < MinFetchTimeout || timeout > MaxFetchTimeout {
		return Config{}, fail(fetchTimeout, "must be a duration between %s and %s", MinFetchTimeout, MaxFetchTimeout)
	}
	cfg.FetchTimeout = timeout

	// First administrator.
	if !validUsername(adminUser.raw) {
		return Config{}, fail(adminUser, "invalid username")
	}
	cfg.AdminUser = adminUser.raw
	cfg.AdminPassword = getenv(EnvAdminPassword)

	// Server key. The value never appears in an error message.
	cfg.SecretKey = secretKey.raw
	if cfg.SecretKey != "" && !validSecretKey(cfg.SecretKey) {
		return Config{}, fail(secretKey, "must be %d bytes encoded in base64", SecretKeyBytes)
	}
	cfg.SecretKeyFile = secretKeyFile.raw
	if cfg.SecretKeyFile == "" {
		cfg.SecretKeyFile = filepath.Join(cfg.DataDir, SecretKeyFileName)
	}

	// Offline recovery commands.
	var commands []*option
	for _, o := range []*option{unlockUser, resetPassword} {
		if set[o.flag] {
			commands = append(commands, o)
		}
	}
	if len(commands) > 1 {
		return Config{}, fmt.Errorf("config: -%s and -%s cannot be combined", unlockUser.flag, resetPassword.flag)
	}
	if len(commands) == 1 {
		o := commands[0]
		for _, other := range []string{"healthcheck", "version"} {
			if set[other] {
				return Config{}, fmt.Errorf("config: -%s cannot be combined with -%s", o.flag, other)
			}
		}
		if !validUsername(o.raw) {
			return Config{}, fail(o, "invalid username")
		}
		cfg.AdminCommand = &AdminCommand{Kind: o.flag, Username: o.raw}
	}
	return cfg, nil
}

// EnvLockedSettings returns the system setting keys fixed by environment
// variables or flags, for example ["allowPrivateFetch"] when
// APP_ALLOW_PRIVATE_FETCH is set. It never returns nil.
func (c Config) EnvLockedSettings() []string {
	locked := []string{}
	if c.AllowPrivateFetch != nil {
		locked = append(locked, settingAllowPrivateFetch)
	}
	return locked
}

// checkAddr accepts "host:port" and ":port" with a numeric port 0-65535.
func checkAddr(s string) error {
	_, port, err := net.SplitHostPort(s)
	if err != nil {
		return errors.New("must be host:port or :port")
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return errors.New("port must be a number between 0 and 65535")
	}
	return nil
}

// parseBaseURL accepts an http or https origin with an empty or "/" path and
// no query, fragment or credentials. The result has an empty path.
func parseBaseURL(s string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil, errors.New("must be an absolute http or https URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("scheme must be http or https")
	}
	if u.Opaque != "" || u.Host == "" || u.Hostname() == "" {
		return nil, errors.New("must be an absolute http or https URL with a host")
	}
	if u.User != nil {
		return nil, errors.New("must not contain credentials")
	}
	if u.Path != "" && u.Path != "/" {
		return nil, errors.New("path must be empty or /")
	}
	if u.RawQuery != "" || u.ForceQuery {
		return nil, errors.New("must not contain a query")
	}
	if u.Fragment != "" || strings.Contains(s, "#") {
		return nil, errors.New("must not contain a fragment")
	}
	if port := u.Port(); port != "" || strings.HasSuffix(u.Host, ":") {
		if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
			return nil, errors.New("port must be a number between 1 and 65535")
		}
	}
	return &url.URL{Scheme: u.Scheme, Host: strings.ToLower(u.Host)}, nil
}

// parseProxies parses a comma-separated list of IP addresses and CIDR
// ranges. A single address becomes a /32 or /128 prefix; IPv4-mapped IPv6
// addresses are unmapped, like the client address (6.2).
func parseProxies(s string) ([]netip.Prefix, error) {
	prefixes := []netip.Prefix{}
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			p, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("invalid IP address or CIDR %q", item)
			}
			if p.Addr().Is4In6() && p.Bits() >= 96 {
				p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
			}
			prefixes = append(prefixes, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(item)
		if err != nil || a.Zone() != "" {
			return nil, fmt.Errorf("invalid IP address or CIDR %q", item)
		}
		a = a.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(a, a.BitLen()))
	}
	return prefixes, nil
}

// parseBool accepts true and false in any letter case.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, errors.New("must be true or false")
}

// parseLogLevel accepts debug, info, warn and error in any letter case.
func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, errors.New("must be debug, info, warn or error")
}

// validSecretKey reports whether s decodes to SecretKeyBytes bytes in one of
// the base64 alphabets. The keyring (internal/auth) performs the final check.
func validSecretKey(s string) bool {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == SecretKeyBytes {
			return true
		}
	}
	return false
}
