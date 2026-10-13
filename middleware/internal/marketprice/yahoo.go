package marketprice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const chartBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart/"

type YahooClient struct {
	httpClient *http.Client
	// For endpoints that need a cookie and crumb (see dividends.go).
	session *yahooSession
}

func NewYahooClient() *YahooClient {
	return &YahooClient{httpClient: &http.Client{Timeout: 15 * time.Second}, session: newYahooSession()}
}

// DailyClose is one day's closing price in the instrument's quote currency.
type DailyClose struct {
	Date     time.Time
	Close    float64
	Currency string
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency string `json:"currency"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// FetchDailyCloses retrieves daily closing prices for symbol between from
// and to (inclusive), skipping non-trading days (weekends/holidays) as
// Yahoo simply omits them from the response.
func (c *YahooClient) FetchDailyCloses(ctx context.Context, symbol string, from, to time.Time) ([]DailyClose, error) {
	reqURL := fmt.Sprintf("%s%s?%s", chartBaseURL, url.PathEscape(symbol), url.Values{
		"period1":  {fmt.Sprintf("%d", from.Unix())},
		"period2":  {fmt.Sprintf("%d", to.Unix())},
		"interval": {"1d"},
	}.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	// Yahoo's chart endpoint rejects requests with no User-Agent header.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; investinsight/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting chart data for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	var parsed chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding chart response for %s: %w", symbol, err)
	}

	if parsed.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo finance error for %s: %s", symbol, parsed.Chart.Error.Description)
	}
	if len(parsed.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data returned for %s", symbol)
	}

	result := parsed.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data returned for %s", symbol)
	}

	closes := result.Indicators.Quote[0].Close
	currency := result.Meta.Currency

	points := make([]DailyClose, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		if i >= len(closes) || closes[i] == nil {
			continue // market holiday/no trade recorded for this timestamp
		}
		points = append(points, DailyClose{
			Date:     time.Unix(ts, 0).UTC().Truncate(24 * time.Hour),
			Close:    *closes[i],
			Currency: currency,
		})
	}

	return points, nil
}
