package irtop

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(cfg Config) *Client {
	client, err := NewClientChecked(cfg)
	if err != nil {
		return &Client{baseURL: strings.TrimRight(cfg.Server, "/"), http: &http.Client{Timeout: 10 * time.Second}}
	}
	return client
}

func NewClientChecked(cfg Config) (*Client, error) {
	cfg.CAFile = expandPath(cfg.CAFile)
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig, err := TLSConfig(cfg)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	return &Client{
		baseURL: strings.TrimRight(cfg.Server, "/"),
		token:   cfg.Token,
		http:    &http.Client{Timeout: 10 * time.Second, Transport: otelhttp.NewTransport(transport)},
	}, nil
}

func ValidateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Server) == "" {
		return fmt.Errorf("server URL is required. For local development use http://localhost:8443. For production use https://ironroot.example.com:8443")
	}
	u, err := url.Parse(cfg.Server)
	if err != nil {
		return fmt.Errorf("invalid server URL %q: %w", cfg.Server, err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("server URL %q is missing a scheme. Use http://localhost:8443 for local development or https://ironroot.example.com:8443 for production", cfg.Server)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("server URL scheme %q is not supported. Use http or https", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("server URL %q is missing a host", cfg.Server)
	}
	return nil
}

func TLSConfig(cfg Config) (*tls.Config, error) {
	if cfg.InsecureSkipVerify {
		return &tls.Config{InsecureSkipVerify: true}, nil //nolint:gosec
	}
	if cfg.CAFile == "" {
		return nil, nil
	}
	pemBytes, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file %q: %w", cfg.CAFile, err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if pool == nil {
		pool = x509.NewCertPool()
	}
	if ok := pool.AppendCertsFromPEM(pemBytes); !ok {
		return nil, fmt.Errorf("failed to load CA certificates from %q", cfg.CAFile)
	}
	return &tls.Config{RootCAs: pool}, nil
}

func (c *Client) Snapshot(ctx context.Context) (Snapshot, error) {
	var out Snapshot
	if err := c.get(ctx, "/v1/status/overview", &out.Overview); err != nil {
		return out, err
	}
	_ = c.get(ctx, "/v1/status/server", &out.Server)
	_ = c.get(ctx, "/v1/status/ca", &out.CA)
	_ = c.get(ctx, "/v1/status/certificates", &out.Certificates)
	_ = c.get(ctx, "/v1/status/enrollments", &out.Enrollments)
	_ = c.get(ctx, "/v1/status/tokens", &out.Tokens)
	_ = c.get(ctx, "/v1/status/security", &out.Security)
	_ = c.get(ctx, "/v1/status/telemetry", &out.Telemetry)
	_ = c.get(ctx, "/v1/audit/recent", &out.Audit)
	out.UpdatedAt = time.Now()
	return out, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return FriendlyConnectionError(c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func FriendlyConnectionError(server string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") {
		return fmt.Errorf(`could not connect to IronRoot API.

The server URL uses HTTPS:
%s

But the server appears to be responding with plain HTTP.

For local development, try:
irtop --server http://localhost:8443

For production, enable TLS on ironroot-server and use:
irtop --server https://<host>:8443 --ca-file ./root-ca.crt

Do not use --insecure-skip-verify unless you understand the risk`, server)
	}
	return err
}

func MaskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
