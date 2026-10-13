package marketprice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Yahoo's quoteSummary endpoint, unlike chart, needs a session: a cookie
// from fc.yahoo.com plus a "crumb" token tied to it. It's unofficial and
// can change without notice, so callers treat failures as "unknown".
const (
	sessionCookieURL = "https://fc.yahoo.com"
	crumbURL         = "https://query2.finance.yahoo.com/v1/test/getcrumb"
	quoteSummaryURL  = "https://query2.finance.yahoo.com/v10/finance/quoteSummary/"
	// The session endpoints reject the plain client User-Agent.
	browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36"
)

var errUnauthorized = errors.New("yahoo session rejected")

// DividendDates is a company's latest dividend dates as Yahoo lists them.
// Either may be nil: UK listings often have no payment date.
type DividendDates struct {
	ExDate  *time.Time
	PayDate *time.Time
}

type yahooSession struct {
	mu     sync.Mutex
	client *http.Client
	crumb  string
}

func newYahooSession() *yahooSession {
	jar, _ := cookiejar.New(nil)
	return &yahooSession{client: &http.Client{Timeout: 15 * time.Second, Jar: jar}}
}

// getCrumb returns the session crumb, starting a session if there isn't
// one (or renew is set, after Yahoo rejected the current one).
func (s *yahooSession) getCrumb(ctx context.Context, renew bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.crumb != "" && !renew {
		return s.crumb, nil
	}

	jar, _ := cookiejar.New(nil)
	s.client.Jar = jar
	// fc.yahoo.com answers 404 but still sets the session cookie.
	if resp, err := s.get(ctx, sessionCookieURL); err != nil {
		return "", fmt.Errorf("starting yahoo session: %w", err)
	} else {
		resp.Body.Close()
	}

	resp, err := s.get(ctx, crumbURL)
	if err != nil {
		return "", fmt.Errorf("fetching yahoo crumb: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", fmt.Errorf("reading yahoo crumb: %w", err)
	}
	crumb := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK || crumb == "" || strings.ContainsAny(crumb, "<{ ") {
		return "", fmt.Errorf("yahoo crumb request failed: status %d", resp.StatusCode)
	}
	s.crumb = crumb
	return crumb, nil
}

func (s *yahooSession) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	return s.client.Do(req)
}

type quoteSummaryResponse struct {
	QuoteSummary struct {
		Result []struct {
			CalendarEvents struct {
				ExDividendDate *struct {
					Raw int64 `json:"raw"`
				} `json:"exDividendDate"`
				DividendDate *struct {
					Raw int64 `json:"raw"`
				} `json:"dividendDate"`
			} `json:"calendarEvents"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"quoteSummary"`
}

// FetchDividendDates returns symbol's latest ex-dividend and payment dates
// from Yahoo's calendar events, renewing the session once if it expired.
func (c *YahooClient) FetchDividendDates(ctx context.Context, symbol string) (*DividendDates, error) {
	dates, err := c.fetchDividendDates(ctx, symbol, false)
	if errors.Is(err, errUnauthorized) {
		dates, err = c.fetchDividendDates(ctx, symbol, true)
	}
	return dates, err
}

func (c *YahooClient) fetchDividendDates(ctx context.Context, symbol string, renew bool) (*DividendDates, error) {
	crumb, err := c.session.getCrumb(ctx, renew)
	if err != nil {
		return nil, err
	}
	reqURL := quoteSummaryURL + url.PathEscape(symbol) + "?" + url.Values{
		"modules": {"calendarEvents"},
		"crumb":   {crumb},
	}.Encode()

	resp, err := c.session.get(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("requesting dividend dates for %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errUnauthorized
	}

	var parsed quoteSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding dividend dates for %s: %w", symbol, err)
	}
	if parsed.QuoteSummary.Error != nil {
		return nil, fmt.Errorf("yahoo finance error for %s: %s", symbol, parsed.QuoteSummary.Error.Description)
	}
	if len(parsed.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("no calendar events returned for %s", symbol)
	}

	events := parsed.QuoteSummary.Result[0].CalendarEvents
	dates := &DividendDates{}
	if events.ExDividendDate != nil && events.ExDividendDate.Raw > 0 {
		d := time.Unix(events.ExDividendDate.Raw, 0).UTC().Truncate(24 * time.Hour)
		dates.ExDate = &d
	}
	if events.DividendDate != nil && events.DividendDate.Raw > 0 {
		d := time.Unix(events.DividendDate.Raw, 0).UTC().Truncate(24 * time.Hour)
		dates.PayDate = &d
	}
	return dates, nil
}

// dividendDatesTTL bounds how long Yahoo's dividend dates are reused.
// Announcements change rarely, and the calendar asks for every holding.
const dividendDatesTTL = 12 * time.Hour

type dividendDatesEntry struct {
	dates   *DividendDates // nil when the lookup failed
	fetched time.Time
}

type dividendDatesCache struct {
	mu      sync.Mutex
	entries map[string]dividendDatesEntry
}

// DividendDates returns inst's latest dividend dates from Yahoo, trying the
// LSE listing if the default symbol has none (as price lookups do).
// Results, including failures, are cached for dividendDatesTTL so a broken
// symbol isn't retried on every page load.
func (s *Service) DividendDates(ctx context.Context, inst Instrument) (*DividendDates, error) {
	symbol := YahooSymbol(inst.Ticker, inst.BaseCurrency, inst.ISIN)

	s.dividendDates.mu.Lock()
	entry, ok := s.dividendDates.entries[symbol]
	s.dividendDates.mu.Unlock()
	if ok && time.Since(entry.fetched) < dividendDatesTTL {
		if entry.dates == nil {
			return nil, fmt.Errorf("no dividend dates for %s (cached)", symbol)
		}
		return entry.dates, nil
	}

	dates, err := s.yahoo.FetchDividendDates(ctx, symbol)
	if (err != nil || dates.ExDate == nil) && !strings.HasSuffix(symbol, ".L") {
		if lse, lseErr := s.yahoo.FetchDividendDates(ctx, symbol+".L"); lseErr == nil && lse.ExDate != nil {
			dates, err = lse, nil
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err() // a cancelled request says nothing about the symbol
	}

	entry = dividendDatesEntry{fetched: time.Now()}
	if err == nil {
		entry.dates = dates
	}
	s.dividendDates.mu.Lock()
	s.dividendDates.entries[symbol] = entry
	s.dividendDates.mu.Unlock()
	return dates, err
}
