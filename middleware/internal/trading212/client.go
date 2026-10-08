package trading212

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Credentials are a Trading 212 API key/secret pair, sent as HTTP Basic auth.
type Credentials struct {
	APIKey    string
	APISecret string
}

// ErrUnauthorized means Trading 212 rejected the credentials (401/403).
var ErrUnauthorized = errors.New("trading 212 rejected the API key/secret (check the key and its permissions)")

// ErrRateLimited means Trading 212 returned 429 for the call.
var ErrRateLimited = errors.New("trading 212 rate limit exceeded")

// Client is a minimal Trading 212 public API client covering what sync needs:
// the account summary (to validate credentials) and CSV history exports.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing Trading 212 base URL: %w", err)
	}
	return &Client{baseURL: u, httpClient: &http.Client{Timeout: 60 * time.Second}}, nil
}

// AccountSummary is the subset of GET /equity/account/summary sync uses.
type AccountSummary struct {
	ID       int64  `json:"id"`
	Currency string `json:"currency"`
}

// Report is one entry of GET /equity/history/exports.
type Report struct {
	ReportID     int64  `json:"reportId"`
	Status       string `json:"status"`
	DownloadLink string `json:"downloadLink"`
}

const (
	ReportFinished = "Finished"
	ReportFailed   = "Failed"
	ReportCanceled = "Canceled"
)

// AccountSummary fetches the account's id and primary currency. Rate limit:
// 1 req / 5s.
func (c *Client) AccountSummary(ctx context.Context, creds Credentials) (*AccountSummary, error) {
	var out AccountSummary
	if err := c.do(ctx, creds, http.MethodGet, "/api/v0/equity/account/summary", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RequestExport asks Trading 212 to generate a CSV of every order, dividend,
// interest payment and cash transaction between from and to. Rate limit:
// 1 req / 30s.
func (c *Client) RequestExport(ctx context.Context, creds Credentials, from, to time.Time) (int64, error) {
	body := map[string]any{
		"dataIncluded": map[string]bool{
			"includeDividends":    true,
			"includeInterest":     true,
			"includeOrders":       true,
			"includeTransactions": true,
		},
		"timeFrom": from.UTC().Format(time.RFC3339),
		"timeTo":   to.UTC().Format(time.RFC3339),
	}
	var out struct {
		ReportID int64 `json:"reportId"`
	}
	if err := c.do(ctx, creds, http.MethodPost, "/api/v0/equity/history/exports", body, &out); err != nil {
		return 0, err
	}
	return out.ReportID, nil
}

// ListExports returns every requested report and its status. Rate limit:
// 1 req / 1m.
func (c *Client) ListExports(ctx context.Context, creds Credentials) ([]Report, error) {
	var out []Report
	if err := c.do(ctx, creds, http.MethodGet, "/api/v0/equity/history/exports", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Download fetches a finished report's CSV. The link is a pre-signed URL, so
// no Trading 212 credentials are sent with it.
func (c *Client) Download(ctx context.Context, link string) ([]byte, error) {
	u, err := c.baseURL.Parse(link) // resolves a relative link against the API host
	if err != nil {
		return nil, fmt.Errorf("parsing download link: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading report: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading report: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 100<<20))
}

func (c *Client) do(ctx context.Context, creds Credentials, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL.JoinPath(path).String(), reqBody)
	if err != nil {
		return err
	}
	req.SetBasicAuth(creds.APIKey, creds.APISecret)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling trading 212 %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrUnauthorized
	case resp.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("trading 212 %s %s: HTTP %d: %s", method, path, resp.StatusCode, bytes.TrimSpace(snippet))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding trading 212 %s %s response: %w", method, path, err)
	}
	return nil
}
