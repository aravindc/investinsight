package portfolio

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"investinsight/middleware/internal/marketprice"
)

type DividendStatus string

const (
	// DividendPaid was received, per the ledger.
	DividendPaid DividendStatus = "paid"
	// DividendAnnounced has an upcoming ex-dividend or payment date on Yahoo.
	DividendAnnounced DividendStatus = "announced"
	// DividendEstimated repeats a payment from the last year, a year on.
	DividendEstimated DividendStatus = "estimated"
)

// DividendEvent is one dividend payment on the calendar. Date is the
// payment date (actual, announced or estimated). AmountGBP is nil for an
// announcement with no past payment to estimate its amount from.
type DividendEvent struct {
	Date         string           `json:"date"`
	InstrumentID uuid.UUID        `json:"instrumentId"`
	Ticker       string           `json:"ticker"`
	Name         string           `json:"name"`
	AmountGBP    *decimal.Decimal `json:"amountGbp"`
	Status       DividendStatus   `json:"status"`
	// ExDate is the announced ex-dividend date, when known.
	ExDate *string `json:"exDate"`
}

// DividendCalendar spans the year before and after Today.
type DividendCalendar struct {
	Today  string          `json:"today"`
	From   string          `json:"from"`
	To     string          `json:"to"`
	Events []DividendEvent `json:"events"`
}

// announcementWindow is how far an announced ex-date may sit from an
// estimated payment and still be taken as that same dividend.
const announcementWindow = 60 * 24 * time.Hour

// dividendLookupConcurrency bounds parallel Yahoo lookups.
const dividendLookupConcurrency = 8

const isoDate = "2006-01-02"

// DividendCalendar combines dividends paid in the last year with the next
// year's: each past payment from a holding still held, repeated a year on
// and scaled to today's shares, with Yahoo's announced dates taking over
// where a holding has one.
func (s *SummaryService) DividendCalendar(ctx context.Context, scope Scope) (*DividendCalendar, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	from := today.AddDate(-1, 0, 0)

	payments, err := s.repo.DividendPayments(ctx, scope, from)
	if err != nil {
		return nil, err
	}
	deltas, err := s.repo.ShareDeltasForCurrentHoldings(ctx, scope)
	if err != nil {
		return nil, err
	}
	holdings, err := s.repo.CurrentHoldings(ctx, scope, NoFilter)
	if err != nil {
		return nil, err
	}

	announced := s.announcedDividends(ctx, holdings, today)
	return buildDividendCalendar(today, payments, deltas, holdings, announced), nil
}

// announcedDividends looks up Yahoo's dividend dates for each holding,
// keeping only those still to come. A failed lookup just means no
// announcement for that holding.
func (s *SummaryService) announcedDividends(ctx context.Context, holdings []CurrentHolding, today time.Time) map[uuid.UUID]marketprice.DividendDates {
	announced := make(map[uuid.UUID]marketprice.DividendDates)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, dividendLookupConcurrency)
	for _, h := range holdings {
		wg.Add(1)
		sem <- struct{}{}
		go func(h CurrentHolding) {
			defer wg.Done()
			defer func() { <-sem }()
			dates, err := s.prices.DividendDates(ctx, marketprice.Instrument{
				ID: h.InstrumentID, Ticker: h.Ticker, BaseCurrency: h.BaseCurrency, ISIN: h.ISIN,
			})
			if err != nil {
				s.logger.Debug("no dividend dates", zap.String("ticker", h.Ticker), zap.Error(err))
				return
			}
			upcoming := (dates.ExDate != nil && dates.ExDate.After(today)) ||
				(dates.PayDate != nil && dates.PayDate.After(today))
			if upcoming {
				mu.Lock()
				announced[h.InstrumentID] = *dates
				mu.Unlock()
			}
		}(h)
	}
	wg.Wait()
	return announced
}

// sharesOn returns shares held at the end of day, per the ledger deltas.
func sharesOn(deltas []ShareDelta, day time.Time) decimal.Decimal {
	held := decimal.Zero
	for _, d := range deltas {
		if d.Date.UTC().After(day) {
			break
		}
		held = held.Add(d.SharesDelta)
	}
	return held
}

func buildDividendCalendar(
	today time.Time,
	payments []DividendPayment,
	deltas []ShareDelta,
	holdings []CurrentHolding,
	announced map[uuid.UUID]marketprice.DividendDates,
) *DividendCalendar {
	until := today.AddDate(1, 0, 0)

	deltasBy := make(map[uuid.UUID][]ShareDelta)
	for _, d := range deltas {
		deltasBy[d.InstrumentID] = append(deltasBy[d.InstrumentID], d)
	}
	for id := range deltasBy {
		ds := deltasBy[id]
		sort.Slice(ds, func(i, j int) bool { return ds[i].Date.Before(ds[j].Date) })
	}
	held := make(map[uuid.UUID]CurrentHolding, len(holdings))
	for _, h := range holdings {
		held[h.InstrumentID] = h
	}

	type event struct {
		DividendEvent
		date time.Time
	}
	var events []event
	add := func(e DividendEvent, date time.Time) {
		e.Date = date.Format(isoDate)
		events = append(events, event{e, date})
	}

	estimatesBy := make(map[uuid.UUID][]int) // indexes into events
	for _, p := range payments {
		amount := p.AmountGBP.Round(2)
		add(DividendEvent{
			InstrumentID: p.InstrumentID, Ticker: p.Ticker, Name: p.Name,
			AmountGBP: &amount, Status: DividendPaid,
		}, p.Date)

		h, stillHeld := held[p.InstrumentID]
		next := p.Date.AddDate(1, 0, 0)
		if !stillHeld || !next.After(today) || next.After(until) {
			continue
		}
		then := sharesOn(deltasBy[p.InstrumentID], p.Date)
		if !then.IsPositive() {
			continue
		}
		estimate := p.AmountGBP.Mul(h.NetShares).Div(then).Round(2)
		estimatesBy[p.InstrumentID] = append(estimatesBy[p.InstrumentID], len(events))
		add(DividendEvent{
			InstrumentID: p.InstrumentID, Ticker: h.Ticker, Name: h.Name,
			AmountGBP: &estimate, Status: DividendEstimated,
		}, next)
	}

	// An announcement takes over the estimate nearest its ex-date, moving it
	// to the announced payment date when Yahoo has one.
	for _, h := range holdings {
		dates, ok := announced[h.InstrumentID]
		if !ok {
			continue
		}
		anchor := dates.ExDate
		if anchor == nil {
			anchor = dates.PayDate
		}
		var exDate *string
		if dates.ExDate != nil {
			s := dates.ExDate.Format(isoDate)
			exDate = &s
		}

		match := -1
		for _, i := range estimatesBy[h.InstrumentID] {
			gap := events[i].date.Sub(*anchor).Abs()
			if gap <= announcementWindow && (match < 0 || gap < events[match].date.Sub(*anchor).Abs()) {
				match = i
			}
		}

		date := *anchor
		if dates.PayDate != nil {
			date = *dates.PayDate
		} else if match >= 0 {
			date = events[match].date // keep the estimated payment date
		}
		if date.After(until) || !date.After(today) {
			continue
		}
		if match >= 0 {
			events[match].Status = DividendAnnounced
			events[match].ExDate = exDate
			events[match].date = date
			events[match].Date = date.Format(isoDate)
			continue
		}
		add(DividendEvent{
			InstrumentID: h.InstrumentID, Ticker: h.Ticker, Name: h.Name,
			Status: DividendAnnounced, ExDate: exDate,
		}, date)
	}

	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].date.Equal(events[j].date) {
			return events[i].date.Before(events[j].date)
		}
		return events[i].Ticker < events[j].Ticker
	})
	out := make([]DividendEvent, len(events))
	for i, e := range events {
		out[i] = e.DividendEvent
	}
	return &DividendCalendar{
		Today:  today.Format(isoDate),
		From:   today.AddDate(-1, 0, 0).Format(isoDate),
		To:     until.Format(isoDate),
		Events: out,
	}
}
