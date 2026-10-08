package cashisa

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// ukLocation decides what "today" is: interest is credited on UK calendar days.
var ukLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func today() time.Time {
	return civil(time.Now().In(ukLocation))
}

// Scope is one account or every account in a household — the same "scope"
// values the portfolio endpoints take.
type Scope struct {
	AccountID   uuid.UUID
	HouseholdID uuid.UUID
	Household   bool
}

func parseScope(raw string) (*Scope, error) {
	if raw == "" {
		return nil, nil // all accounts
	}
	if id, ok := strings.CutPrefix(raw, "household:"); ok {
		hid, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid household id: %w", err)
		}
		return &Scope{HouseholdID: hid, Household: true}, nil
	}
	aid, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid scope: %w", err)
	}
	return &Scope{AccountID: aid}, nil
}

type Handler struct {
	repo *Repository
	log  *zap.Logger
}

func NewHandler(repo *Repository, log *zap.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/cash-isas", h.list)
	router.PUT("/accounts/:id/cash-isa", h.save)
	router.POST("/accounts/:id/cash-isa/rates", h.setRate)
	router.DELETE("/accounts/:id/cash-isa", h.delete)
}

// CashISA is one account's Cash ISA valued as of today.
type CashISA struct {
	AccountID        uuid.UUID       `json:"accountId"`
	AccountName      string          `json:"accountName"`
	Balance          decimal.Decimal `json:"balanceGbp"`
	BalanceDate      string          `json:"balanceDate"`
	Value            decimal.Decimal `json:"valueGbp"`
	AccruedThisMonth decimal.Decimal `json:"accruedThisMonthGbp"`
	InterestCredited decimal.Decimal `json:"interestCreditedGbp"`
	CurrentRate      decimal.Decimal `json:"currentRatePercent"`
	NextCreditDate   string          `json:"nextCreditDate"`
	Rates            []rateJSON      `json:"rates"`
}

type rateJSON struct {
	EffectiveFrom string          `json:"effectiveFrom"`
	RatePercent   decimal.Decimal `json:"ratePercent"`
}

// Summary totals the Cash ISAs in scope.
type Summary struct {
	TotalValue            decimal.Decimal `json:"totalValueGbp"`
	TotalAccruedThisMonth decimal.Decimal `json:"totalAccruedThisMonthGbp"`
	AsOf                  string          `json:"asOf"`
	CashISAs              []CashISA       `json:"cashIsas"`
}

const dateLayout = "2006-01-02"

func (h *Handler) list(c *gin.Context) {
	scope, err := parseScope(c.Query("scope"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isas, err := h.repo.List(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to list cash ISAs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load Cash ISAs"})
		return
	}

	asOf := today()
	out := Summary{AsOf: asOf.Format(dateLayout), CashISAs: []CashISA{}}
	for _, s := range isas {
		v := Value(s.Balance, s.BalanceDate, s.Rates, asOf)
		item := CashISA{
			AccountID:        s.AccountID,
			AccountName:      s.AccountName,
			Balance:          s.Balance,
			BalanceDate:      s.BalanceDate.Format(dateLayout),
			Value:            v.Value,
			AccruedThisMonth: v.AccruedThisMonth,
			InterestCredited: v.InterestCredited,
			CurrentRate:      v.CurrentRate,
			NextCreditDate:   v.NextCreditDate.Format(dateLayout),
			Rates:            []rateJSON{},
		}
		for _, r := range s.Rates {
			item.Rates = append(item.Rates, rateJSON{r.EffectiveFrom.Format(dateLayout), r.AnnualPercent})
		}
		out.TotalValue = out.TotalValue.Add(v.Value)
		out.TotalAccruedThisMonth = out.TotalAccruedThisMonth.Add(v.AccruedThisMonth)
		out.CashISAs = append(out.CashISAs, item)
	}
	c.JSON(http.StatusOK, out)
}

type saveRequest struct {
	Balance     string `json:"balance"`
	BalanceDate string `json:"balanceDate"`
	RatePercent string `json:"ratePercent"`
}

func (h *Handler) save(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	var req saveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	balance, err := decimal.NewFromString(strings.TrimSpace(req.Balance))
	if err != nil || balance.IsNegative() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "balance must be a number of 0 or more"})
		return
	}
	date, ok := parseDate(c, req.BalanceDate, "balanceDate")
	if !ok {
		return
	}
	rate, ok := parseRate(c, req.RatePercent)
	if !ok {
		return
	}

	err = h.repo.Save(c.Request.Context(), accountID, balance.Round(2), date, rate)
	var pqErr *pq.Error
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.As(err, &pqErr) && pqErr.Constraint == "cash_isas_account_id_fkey":
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
	default:
		h.log.Error("failed to save cash ISA", zap.String("accountId", accountID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save Cash ISA"})
	}
}

type rateRequest struct {
	RatePercent   string `json:"ratePercent"`
	EffectiveFrom string `json:"effectiveFrom"`
}

func (h *Handler) setRate(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	var req rateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	date, ok := parseDate(c, req.EffectiveFrom, "effectiveFrom")
	if !ok {
		return
	}
	rate, ok := parseRate(c, req.RatePercent)
	if !ok {
		return
	}

	found, err := h.repo.SetRate(c.Request.Context(), accountID, date, rate)
	switch {
	case err != nil:
		h.log.Error("failed to set cash ISA rate", zap.String("accountId", accountID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save rate"})
	case !found:
		c.JSON(http.StatusNotFound, gin.H{"error": "this account has no Cash ISA"})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handler) delete(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	if err := h.repo.Delete(c.Request.Context(), accountID); err != nil {
		h.log.Error("failed to delete cash ISA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove Cash ISA"})
		return
	}
	c.Status(http.StatusNoContent)
}

func parseAccountID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return uuid.Nil, false
	}
	return id, true
}

// parseDate reads a YYYY-MM-DD date that isn't in the future.
func parseDate(c *gin.Context, raw, field string) (time.Time, bool) {
	d, err := time.Parse(dateLayout, strings.TrimSpace(raw))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": field + " must be a date (YYYY-MM-DD)"})
		return time.Time{}, false
	}
	if d.After(today()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": field + " can't be in the future"})
		return time.Time{}, false
	}
	return d, true
}

func parseRate(c *gin.Context, raw string) (decimal.Decimal, bool) {
	rate, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || rate.IsNegative() || rate.GreaterThan(hundred) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ratePercent must be between 0 and 100"})
		return decimal.Zero, false
	}
	return rate.Round(3), true
}
