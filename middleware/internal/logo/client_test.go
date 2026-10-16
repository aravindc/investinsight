package logo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp: connection refused")
}

func TestFetchLogoErrorOmitsToken(t *testing.T) {
	const token = "pk_secret_token_value"
	c := NewClient(token)
	c.httpClient.Transport = failingTransport{}

	_, err := c.FetchLogo(context.Background(), "LLOY.L")
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaks the token: %v", err)
	}
	if !strings.Contains(err.Error(), "img.logo.dev/ticker/LLOY.L") {
		t.Errorf("error should still name the request: %v", err)
	}
}

// recordingTransport answers every request with status and remembers the
// last request.
type recordingTransport struct {
	status int
	last   *http.Request
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.last = req
	return &http.Response{
		StatusCode: t.status,
		Header:     http.Header{"Content-Type": {"image/jpeg"}},
		Body:       io.NopCloser(strings.NewReader("img")),
		Request:    req,
	}, nil
}

func TestFetchLogoAsksFor404InsteadOfPlaceholder(t *testing.T) {
	transport := &recordingTransport{status: http.StatusNotFound}
	c := NewClient("tok")
	c.httpClient.Transport = transport

	_, err := c.FetchLogo(context.Background(), "AAPI.L")

	// Without fallback=404, logo.dev returns 200 with a letter tile that
	// would be stored as the instrument's logo.
	if got := transport.last.URL.Query().Get("fallback"); got != "404" {
		t.Errorf("fallback = %q, want 404", got)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// sequenceTransport answers with each status in turn, repeating the last.
type sequenceTransport struct {
	statuses []int
	calls    int
}

func (t *sequenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	status := t.statuses[min(t.calls, len(t.statuses)-1)]
	t.calls++
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"image/jpeg"}},
		Body:       io.NopCloser(strings.NewReader("img")),
		Request:    req,
	}, nil
}

func TestFetchLogoRetriesWhenRateLimited(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{429, 429, 200}}
	c := NewClient("tok")
	c.httpClient.Transport = transport
	var waits []time.Duration
	c.sleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}

	img, err := c.FetchLogo(context.Background(), "AAPL")

	if err != nil || string(img.Data) != "img" {
		t.Fatalf("FetchLogo = %v, %v; want the image after retrying", img, err)
	}
	if transport.calls != 3 || len(waits) != 2 || waits[0] != time.Second || waits[1] != 2*time.Second {
		t.Errorf("calls = %d, waits = %v; want 3 calls and waits of 1s then 2s", transport.calls, waits)
	}
}

func TestFetchLogoGivesUpAfterRepeatedRateLimits(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{429}}
	c := NewClient("tok")
	c.httpClient.Transport = transport
	c.sleep = func(context.Context, time.Duration) error { return nil }

	_, err := c.FetchLogo(context.Background(), "AAPL")

	if err == nil || errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "429") {
		t.Errorf("err = %v; want a 429 error, not ErrNotFound (which would delete the logo)", err)
	}
	if transport.calls != rateLimitRetries+1 {
		t.Errorf("calls = %d, want %d", transport.calls, rateLimitRetries+1)
	}
}

// pathTransport answers 200 for listed paths and 404 for the rest,
// recording the paths asked for.
type pathTransport struct {
	found map[string]bool
	asked []string
}

func (t *pathTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.asked = append(t.asked, req.URL.Path)
	status := http.StatusNotFound
	if t.found[req.URL.Path] {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"image/jpeg"}},
		Body:       io.NopCloser(strings.NewReader("img")),
		Request:    req,
	}, nil
}

func TestFetchLogoWithLSEFallback(t *testing.T) {
	cases := []struct {
		name      string
		symbol    string
		found     []string
		wantFrom  string
		wantAsked []string
		wantErr   error
	}{
		{"plain symbol found", "AAPL", []string{"/ticker/AAPL"}, "AAPL", []string{"/ticker/AAPL"}, nil},
		{"falls back to the LSE listing", "JEIP", []string{"/ticker/JEIP.L"}, "JEIP.L", []string{"/ticker/JEIP", "/ticker/JEIP.L"}, nil},
		{"neither has one", "JEGP", nil, "JEGP", []string{"/ticker/JEGP", "/ticker/JEGP.L"}, ErrNotFound},
		{"already LSE: no second try", "VUSA.L", nil, "VUSA.L", []string{"/ticker/VUSA.L"}, ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &pathTransport{found: map[string]bool{}}
			for _, p := range tc.found {
				transport.found[p] = true
			}
			c := NewClient("tok")
			c.httpClient.Transport = transport

			_, from, err := c.FetchLogoWithLSEFallback(context.Background(), tc.symbol)

			if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
			if from != tc.wantFrom {
				t.Errorf("found under %q, want %q", from, tc.wantFrom)
			}
			if strings.Join(transport.asked, ",") != strings.Join(tc.wantAsked, ",") {
				t.Errorf("asked %v, want %v", transport.asked, tc.wantAsked)
			}
		})
	}
}
