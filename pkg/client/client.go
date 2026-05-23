package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

type EnrollmentRequest struct {
	Token     string `json:"token"`
	Hostname  string `json:"hostname"`
	MachineID string `json:"machine_id"`
	MAC       string `json:"mac,omitempty"`
}

type EnrollmentResponse struct {
	EnrollmentID string `json:"enrollment_id"`
}

type CertificateRequest struct {
	EnrollmentID string   `json:"enrollment_id"`
	CSRPEM       string   `json:"csr_pem"`
	DNSNames     []string `json:"dns_names"`
}

type CertificateResponse struct {
	Serial      string `json:"serial"`
	CertPEM     string `json:"cert_pem"`
	ChainPEM    string `json:"chain_pem"`
	RootPEM     string `json:"root_pem"`
	NotBefore   string `json:"not_before"`
	NotAfter    string `json:"not_after"`
	RenewBefore string `json:"renew_before"`
}

func (c *Client) Enroll(ctx context.Context, req EnrollmentRequest) (EnrollmentResponse, error) {
	var out EnrollmentResponse
	return out, c.post(ctx, "/v1/enroll", req, &out)
}

func (c *Client) RequestCertificate(ctx context.Context, req CertificateRequest) (CertificateResponse, error) {
	var out CertificateResponse
	return out, c.post(ctx, "/v1/certificates/request", req, &out)
}

func (c *Client) RenewCertificate(ctx context.Context, req CertificateRequest) (CertificateResponse, error) {
	var out CertificateResponse
	return out, c.post(ctx, "/v1/certificates/renew", req, &out)
}

func (c *Client) RevokeCertificate(ctx context.Context, serial, reason string) error {
	return c.post(ctx, "/v1/certificates/revoke", map[string]string{"serial": serial, "reason": reason}, nil)
}

func (c *Client) Status(ctx context.Context, serial string) (map[string]any, error) {
	var out map[string]any
	return out, c.get(ctx, "/v1/certificates/status/"+serial, &out)
}

func (c *Client) Audit(ctx context.Context) ([]map[string]any, error) {
	var out []map[string]any
	return out, c.get(ctx, "/v1/audit", &out)
}

func (c *Client) Root(ctx context.Context) (string, error) {
	return c.getRaw(ctx, "/v1/ca/root")
}

func (c *Client) post(ctx context.Context, path string, in, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) getRaw(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(b, out)
}
