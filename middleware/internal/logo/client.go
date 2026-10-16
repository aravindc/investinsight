package logo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const logoDevBaseURL = "https://img.logo.dev/ticker/"

// rateLimitRetries is how many times a rate-limited (429) request is
// retried. Without Retry-After, waits double from rateLimitBackoff:
// 1s, 2s, 4s, 8s.
const (
	rateLimitRetries = 4
	rateLimitBackoff = time.Second
)

// Client fetches ticker logo images from logo.dev.
type Client struct {
	httpClient *http.Client
	token      string
	// sleep waits between rate-limit retries; replaced in tests.
	sleep func(ctx context.Context, d time.Duration) error
}

func NewClient(token string) *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}, token: token, sleep: sleepCtx}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retryAfter reads a Retry-After header given in seconds, else fallback.
func retryAfter(resp *http.Response, fallback time.Duration) time.Duration {
	if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return fallback
}

// Image is a fetched logo's raw bytes and its content type.
type Image struct {
	Data        []byte
	ContentType string
}

// ErrNotFound is returned when logo.dev has no logo for the given symbol.
// Requests ask for a 404 in that case: by default logo.dev answers 200
// with a generated letter tile, which would be stored as the logo.
var ErrNotFound = fmt.Errorf("logo not found")

// FetchLogoWithLSEFallback fetches symbol's logo and, if logo.dev has
// none, tries the London listing (symbol + ".L"), as price lookups do:
// some instruments quoted in USD or EUR are LSE-listed (e.g. JPMorgan's
// JEIP). It returns the symbol the logo was found under.
func (c *Client) FetchLogoWithLSEFallback(ctx context.Context, symbol string) (*Image, string, error) {
	image, err := c.FetchLogo(ctx, symbol)
	if !errors.Is(err, ErrNotFound) || strings.HasSuffix(symbol, ".L") {
		return image, symbol, err
	}
	lse := symbol + ".L"
	image, err = c.FetchLogo(ctx, lse)
	if err != nil {
		return nil, symbol, err
	}
	return image, lse, nil
}

// FetchLogo retrieves the logo image for symbol (e.g. "LLOY.L", "AAPL").
func (c *Client) FetchLogo(ctx context.Context, symbol string) (*Image, error) {
	reqURL := fmt.Sprintf("%s%s?%s", logoDevBaseURL, url.PathEscape(symbol), url.Values{
		"token":    {c.token},
		"fallback": {"404"},
	}.Encode())

	// logo.dev rate-limits bursts (a refresh asks for every instrument),
	// so a 429 waits and tries again rather than failing the instrument.
	var resp *http.Response
	backoff := rateLimitBackoff
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("requesting logo for %s: %w", symbol, redactURL(err, symbol))
		}
		if resp.StatusCode != http.StatusTooManyRequests || attempt == rateLimitRetries {
			break
		}
		wait := retryAfter(resp, backoff)
		resp.Body.Close()
		if err := c.sleep(ctx, wait); err != nil {
			return nil, err
		}
		backoff *= 2
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("logo.dev returned status %d for %s", resp.StatusCode, symbol)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading logo body for %s: %w", symbol, err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	return &Image{Data: data, ContentType: contentType}, nil
}

// redactURL strips the query string (which carries the API token) from the
// URL a transport error reports, so the error can be logged safely.
func redactURL(err error, symbol string) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		urlErr.URL = logoDevBaseURL + url.PathEscape(symbol)
	}
	return err
}
