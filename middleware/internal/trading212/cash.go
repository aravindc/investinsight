package trading212

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// summaryCacheTTL is how long a fetched account summary is reused for cash
// figures. Cash seldom changes, and Trading 212 allows only 1 summary
// request per 5s per account. The entry is dropped early when the account
// syncs or its key changes (see forget), and the response carries its age.
const summaryCacheTTL = time.Hour

// reconcileMaxAge lets Reconcile reuse a summary fetched moments ago (e.g.
// by the dashboard) rather than tripping the 5s rate limit.
const reconcileMaxAge = 5 * time.Second

type cachedSummary struct {
	summary *AccountSummary
	fetched time.Time
}

type summaryCache struct {
	mu      sync.Mutex
	entries map[uuid.UUID]cachedSummary
}

func (c *summaryCache) get(accountID uuid.UUID, maxAge time.Duration) (*AccountSummary, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[accountID]
	if !ok || time.Since(e.fetched) > maxAge {
		return nil, false
	}
	return e.summary, true
}

func (c *summaryCache) set(accountID uuid.UUID, s *AccountSummary) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[uuid.UUID]cachedSummary)
	}
	c.entries[accountID] = cachedSummary{summary: s, fetched: time.Now()}
}

// accountSummary returns the account's Trading 212 summary, reusing one
// fetched within maxAge.
func (s *Service) accountSummary(ctx context.Context, accountID uuid.UUID, maxAge time.Duration) (*AccountSummary, error) {
	if cached, ok := s.summaries.get(accountID, maxAge); ok {
		return cached, nil
	}
	creds, err := s.credentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	summary, err := s.client.AccountSummary(ctx, creds)
	if err != nil {
		return nil, err
	}
	s.summaries.set(accountID, summary)
	return summary, nil
}

// CashScope selects one account, or every account in a household — the
// same "scope" values the portfolio endpoints take.
type CashScope struct {
	AccountID   uuid.UUID
	HouseholdID uuid.UUID
	Household   bool
}

// ParseCashScope reads an account UUID or "household:<uuid>".
func ParseCashScope(raw string) (CashScope, error) {
	if id, ok := strings.CutPrefix(raw, "household:"); ok {
		hid, err := uuid.Parse(id)
		if err != nil {
			return CashScope{}, fmt.Errorf("invalid household id: %w", err)
		}
		return CashScope{HouseholdID: hid, Household: true}, nil
	}
	aid, err := uuid.Parse(raw)
	if err != nil {
		return CashScope{}, fmt.Errorf("invalid scope: %w", err)
	}
	return CashScope{AccountID: aid}, nil
}

// AccountCashIssue names an account in scope whose cash isn't included.
type AccountCashIssue struct {
	AccountID uuid.UUID `json:"accountId"`
	Name      string    `json:"name"`
	Reason    string    `json:"reason"`
}

// CashSummary is Trading 212 cash across the connected accounts in a scope.
// Cash is available-to-trade plus cash reserved for pending orders; cash in
// pies is uninvested money held inside pies.
type CashSummary struct {
	CashGBP          decimal.Decimal    `json:"cashGbp"`
	CashInPiesGBP    decimal.Decimal    `json:"cashInPiesGbp"`
	AccountsIncluded int                `json:"accountsIncluded"`
	AccountsExcluded []AccountCashIssue `json:"accountsExcluded"`
	OldestFetchedAt  *time.Time         `json:"oldestFetchedAt"`
}

// Cash sums live Trading 212 cash for every account in scope. Accounts that
// aren't connected, or whose summary can't be fetched, are listed in
// AccountsExcluded rather than failing the whole request.
func (s *Service) Cash(ctx context.Context, scope CashScope) (*CashSummary, error) {
	accounts, err := s.repo.AccountsInScope(ctx, scope)
	if err != nil {
		return nil, err
	}

	type result struct {
		acct    scopedAccount
		summary *AccountSummary
		err     error
	}
	results := make([]result, len(accounts))
	var wg sync.WaitGroup
	for i, a := range accounts {
		results[i].acct = a
		if !a.Connected {
			continue
		}
		wg.Add(1)
		go func(i int, a scopedAccount) {
			defer wg.Done()
			results[i].summary, results[i].err = s.accountSummary(ctx, a.ID, summaryCacheTTL)
		}(i, a)
	}
	wg.Wait()

	out := &CashSummary{AccountsExcluded: []AccountCashIssue{}}
	for _, r := range results {
		switch {
		case !r.acct.Connected:
			out.AccountsExcluded = append(out.AccountsExcluded, AccountCashIssue{r.acct.ID, r.acct.Name, "not connected to Trading 212"})
		case r.err != nil:
			reason := "couldn't fetch from Trading 212"
			if errors.Is(r.err, ErrUnauthorized) {
				reason = "Trading 212 rejected the stored API key"
			} else if errors.Is(r.err, ErrRateLimited) {
				reason = "Trading 212 rate limit hit; refresh shortly"
			}
			s.logger.Sugar().Warnw("trading 212 cash unavailable", "accountId", r.acct.ID, "error", r.err)
			out.AccountsExcluded = append(out.AccountsExcluded, AccountCashIssue{r.acct.ID, r.acct.Name, reason})
		case !strings.EqualFold(r.summary.Currency, "GBP"):
			out.AccountsExcluded = append(out.AccountsExcluded, AccountCashIssue{r.acct.ID, r.acct.Name, "account currency isn't GBP"})
		default:
			c := r.summary.Cash
			out.CashGBP = out.CashGBP.Add(c.AvailableToTrade).Add(c.ReservedForOrders)
			out.CashInPiesGBP = out.CashInPiesGBP.Add(c.InPies)
			out.AccountsIncluded++
			if e, ok := s.summaries.fetchedAt(r.acct.ID); ok && (out.OldestFetchedAt == nil || e.Before(*out.OldestFetchedAt)) {
				out.OldestFetchedAt = &e
			}
		}
	}
	out.CashGBP = out.CashGBP.Round(2)
	out.CashInPiesGBP = out.CashInPiesGBP.Round(2)
	return out, nil
}

func (c *summaryCache) forget(accountID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, accountID)
}

// ForgetSummary drops the cached Trading 212 summary for accountID, so the
// next cash request fetches fresh figures.
func (s *Service) ForgetSummary(accountID uuid.UUID) {
	s.summaries.forget(accountID)
}

func (c *summaryCache) fetchedAt(accountID uuid.UUID) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[accountID]
	return e.fetched.UTC(), ok
}
