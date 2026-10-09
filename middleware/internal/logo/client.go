package logo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const logoDevBaseURL = "https://img.logo.dev/ticker/"

// Client fetches ticker logo images from logo.dev.
type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(token string) *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}, token: token}
}

// Image is a fetched logo's raw bytes and its content type.
type Image struct {
	Data        []byte
	ContentType string
}

// ErrNotFound is returned when logo.dev has no logo for the given symbol.
var ErrNotFound = fmt.Errorf("logo not found")

// FetchLogo retrieves the logo image for symbol (e.g. "LLOY.L", "AAPL").
func (c *Client) FetchLogo(ctx context.Context, symbol string) (*Image, error) {
	reqURL := fmt.Sprintf("%s%s?%s", logoDevBaseURL, url.PathEscape(symbol), url.Values{
		"token": {c.token},
	}.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting logo for %s: %w", symbol, redactURL(err, symbol))
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
