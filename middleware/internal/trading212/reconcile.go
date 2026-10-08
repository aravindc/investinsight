package trading212

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"investinsight/middleware/internal/portfolio"
)

// HoldingsSummarizer computes the app's own invested and market value; it
// is implemented by *portfolio.SummaryService.
type HoldingsSummarizer interface {
	HoldingsSummary(ctx context.Context, scope portfolio.Scope, filter portfolio.Filter) (*portfolio.HoldingsSummary, error)
}

// reconcileTolerancePercent is how far (relative to Trading 212's figure) the
// app's figure may differ and still count as matching. Market values never
// match exactly: Trading 212 prices in real time, the app from Yahoo closes.
var reconcileTolerancePercent = decimal.NewFromInt(1)

// ReconciliationLine compares one figure between Trading 212 and the app.
type ReconciliationLine struct {
	Key               string           `json:"key"`
	Label             string           `json:"label"`
	Trading212        decimal.Decimal  `json:"trading212"`
	App               decimal.Decimal  `json:"app"`
	Difference        decimal.Decimal  `json:"difference"`        // app − Trading 212
	DifferencePercent *decimal.Decimal `json:"differencePercent"` // of Trading 212's figure; nil when that is zero
	Matches           bool             `json:"matches"`
}

// Reconciliation is Trading 212's account summary set against the app's
// figures for the same account. Cash and realised P/L aren't tracked by the
// app, so they are reported for information only.
type Reconciliation struct {
	AccountID        uuid.UUID            `json:"accountId"`
	T212AccountID    int64                `json:"t212AccountId"`
	Currency         string               `json:"currency"`
	FetchedAt        time.Time            `json:"fetchedAt"`
	TolerancePercent decimal.Decimal      `json:"tolerancePercent"`
	Lines            []ReconciliationLine `json:"lines"`
	UnpricedHoldings int                  `json:"unpricedHoldings"`

	Trading212Info struct {
		TotalValue         decimal.Decimal `json:"totalValue"`
		CashAvailable      decimal.Decimal `json:"cashAvailable"`
		CashInPies         decimal.Decimal `json:"cashInPies"`
		CashReserved       decimal.Decimal `json:"cashReserved"`
		RealizedProfitLoss decimal.Decimal `json:"realizedProfitLoss"`
	} `json:"trading212Info"`
}

// Reconcile fetches the account summary from Trading 212 and compares it with
// the app's holdings for accountID.
func (s *Service) Reconcile(ctx context.Context, accountID uuid.UUID) (*Reconciliation, error) {
	conn, err := s.repo.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrNotConnected
	}

	summary, err := s.accountSummary(ctx, accountID, reconcileMaxAge)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(summary.Currency, "GBP") {
		return nil, fmt.Errorf("%w (this account is in %s)", ErrUnsupportedCurrency, summary.Currency)
	}

	app, err := s.holdings.HoldingsSummary(ctx, portfolio.Scope{AccountID: accountID}, portfolio.NoFilter)
	if err != nil {
		return nil, fmt.Errorf("computing app holdings: %w", err)
	}

	return buildReconciliation(accountID, summary, app, time.Now().UTC()), nil
}

func buildReconciliation(accountID uuid.UUID, t212 *AccountSummary, app *portfolio.HoldingsSummary, now time.Time) *Reconciliation {
	r := &Reconciliation{
		AccountID:        accountID,
		T212AccountID:    t212.ID,
		Currency:         strings.ToUpper(t212.Currency),
		FetchedAt:        now,
		TolerancePercent: reconcileTolerancePercent,
		UnpricedHoldings: app.UnpricedHoldings,
		Lines: []ReconciliationLine{
			compareLine("marketValue", "Investments value", t212.Investments.CurrentValue, app.MarketValueGBP),
			compareLine("costBasis", "Cost basis", t212.Investments.TotalCost, app.InvestedValueGBP),
			compareLine("unrealizedPL", "Unrealised P/L", t212.Investments.UnrealizedProfitLoss, app.MarketValueGBP.Sub(app.InvestedValueGBP)),
		},
	}
	r.Trading212Info.TotalValue = t212.TotalValue
	r.Trading212Info.CashAvailable = t212.Cash.AvailableToTrade
	r.Trading212Info.CashInPies = t212.Cash.InPies
	r.Trading212Info.CashReserved = t212.Cash.ReservedForOrders
	r.Trading212Info.RealizedProfitLoss = t212.Investments.RealizedProfitLoss
	return r
}

func compareLine(key, label string, t212, app decimal.Decimal) ReconciliationLine {
	t212, app = t212.Round(2), app.Round(2)
	diff := app.Sub(t212)
	line := ReconciliationLine{Key: key, Label: label, Trading212: t212, App: app, Difference: diff}

	if t212.IsZero() {
		line.Matches = diff.IsZero()
		return line
	}
	pct := diff.Div(t212.Abs()).Mul(decimal.NewFromInt(100)).Round(2)
	line.DifferencePercent = &pct
	line.Matches = pct.Abs().LessThanOrEqual(reconcileTolerancePercent)
	return line
}
