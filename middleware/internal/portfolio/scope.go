package portfolio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Scope restricts portfolio queries to either a single account or every
// account within a household (the "All accounts" aggregate view).
type Scope struct {
	AccountID     uuid.UUID
	HouseholdID   uuid.UUID
	HouseholdWide bool
}

// parseScope reads the required "scope" query param: either an account UUID,
// or "household:<uuid>" for a household-wide aggregate.
func parseScope(c *gin.Context) (Scope, error) {
	raw := c.Query("scope")
	if raw == "" {
		return Scope{}, errors.New("missing required scope parameter")
	}

	if id, ok := strings.CutPrefix(raw, "household:"); ok {
		hid, err := uuid.Parse(id)
		if err != nil {
			return Scope{}, fmt.Errorf("invalid household id: %w", err)
		}
		return Scope{HouseholdID: hid, HouseholdWide: true}, nil
	}

	aid, err := uuid.Parse(raw)
	if err != nil {
		return Scope{}, fmt.Errorf("invalid account id: %w", err)
	}
	return Scope{AccountID: aid}, nil
}

// buildScopeFilter returns a SQL condition scoping rows of alias to scope
// ("<alias>.account_id = $N" for a single account, or an IN-subquery over
// every account in the household for the aggregate view), plus the single
// arg to bind at argPos. Pass an empty alias for an unqualified "account_id"
// column reference (e.g. inside a subquery with its own FROM).
func buildScopeFilter(scope Scope, alias string, argPos int) (string, any) {
	col := "account_id"
	if alias != "" {
		col = alias + "." + col
	}
	if scope.HouseholdWide {
		return fmt.Sprintf("%s IN (SELECT id FROM accounts WHERE household_id = $%d)", col, argPos), scope.HouseholdID
	}
	return fmt.Sprintf("%s = $%d", col, argPos), scope.AccountID
}

// Filter narrows a Scope-restricted query to specific instruments (resolved
// from the Analytics page's sector/asset multi-select). Active is false
// when no filter was supplied at all (the "instrumentId" query param was
// absent) — callers must behave exactly as if no filter existed: every
// instrument included, cash-only entries (DEPOSIT/WITHDRAWAL/INTEREST, which
// have no instrument_id) included. Active is true with a possibly-empty
// InstrumentIDs when the frontend explicitly narrowed the selection —
// including to a selection that resolves to zero instruments, which must
// legitimately return empty/zero results, not "everything".
type Filter struct {
	Active        bool
	InstrumentIDs []uuid.UUID
}

// NoFilter is the zero-value "no filter active" filter, used by every call
// site outside the Analytics page so their behavior is unchanged.
var NoFilter = Filter{}

// parseInstrumentFilter reads zero or more repeated "instrumentId" query
// params. Returns Filter{} (inactive) if the param was absent entirely.
func parseInstrumentFilter(c *gin.Context) (Filter, error) {
	raw := c.QueryArray("instrumentId")
	if len(raw) == 0 {
		return Filter{}, nil
	}

	ids := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return Filter{}, fmt.Errorf("invalid instrumentId: %w", err)
		}
		ids = append(ids, id)
	}
	return Filter{Active: true, InstrumentIDs: ids}, nil
}

// buildInstrumentFilter returns a SQL condition restricting alias's
// instrument_id to filter.InstrumentIDs, plus its bind arg, when
// filter.Active is true. Returns ("", nil, false) when the filter is
// inactive — callers must skip appending both the condition and the arg
// entirely in that case (an inactive filter means "no restriction," which
// is the absence of a WHERE clause, not an always-true one).
func buildInstrumentFilter(filter Filter, alias string, argPos int) (string, any, bool) {
	if !filter.Active {
		return "", nil, false
	}
	col := "instrument_id"
	if alias != "" {
		col = alias + "." + col
	}
	return fmt.Sprintf("%s = ANY($%d)", col, argPos), pq.Array(filter.InstrumentIDs), true
}
