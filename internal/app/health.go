package app

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/kaszasg/kinex-mark/internal/config"
)

// healthTimeout bounds the whole health probe (TECHNOLOGY.md 4.3).
const healthTimeout = 3 * time.Second

// CheckHealth queries GET /healthz of the running server on cfg.Addr and
// returns the exit code: 0 on 200, otherwise 1 (TECHNOLOGY.md 4.3). An empty
// or 0.0.0.0 host is replaced by 127.0.0.1; in TLS mode it uses https
// without certificate verification, because the request only goes to the
// local server.
func CheckHealth(cfg config.Config) int {
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return 1
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	scheme := "http"
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if cfg.TLSCertFile != "" {
		scheme = "https"
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // loopback health probe only
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: healthTimeout, Transport: transport}
	res, err := client.Get(scheme + "://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
