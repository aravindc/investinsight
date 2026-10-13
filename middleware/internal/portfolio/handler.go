package portfolio

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"investinsight/middleware/internal/xirr"
)

type Handler struct {
	repo    *Repository
	summary *SummaryService
	log     *zap.Logger
}

func NewHandler(repo *Repository, summary *SummaryService, log *zap.Logger) *Handler {
	return &Handler{repo: repo, summary: summary, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/transactions", h.listTransactions)
	router.GET("/portfolio/holdings-summary", h.holdingsSummary)
	router.GET("/portfolio/dividends-summary", h.dividendsSummary)
	router.GET("/portfolio/monthly-dividends", h.monthlyDividends)
	router.GET("/portfolio/invested-value-series", h.investedValueSeries)
	router.GET("/portfolio/market-value-series", h.marketValueSeries)
	router.GET("/portfolio/sector-allocations", h.sectorAllocations)
	router.GET("/portfolio/allocations", h.allocations)
	router.GET("/portfolio/allocation-matrix", h.allocationMatrix)
	router.GET("/portfolio/allocation-series", h.allocationSeries)
	router.GET("/portfolio/dividend-calendar", h.dividendCalendar)
	router.GET("/portfolio/irr", h.portfolioIRR)
	router.GET("/portfolio/passive-income", h.passiveIncome)
	router.GET("/portfolio/asset-categories", h.assetCategories)
	router.GET("/portfolio/holdings-table", h.holdingsTable)
}

func (h *Handler) holdingsTable(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.summary.HoldingsTable(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute holdings table", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load holdings table"})
		return
	}
	if rows == nil {
		rows = []HoldingRow{}
	}
	c.JSON(http.StatusOK, rows)
}

func (h *Handler) assetCategories(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	categories, err := h.summary.AssetCategories(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute asset categories", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load asset categories"})
		return
	}
	if categories == nil {
		categories = []AssetCategory{}
	}
	c.JSON(http.StatusOK, categories)
}

func (h *Handler) listTransactions(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	transactions, err := h.repo.ListTransactions(c.Request.Context(), 200, scope)
	if err != nil {
		h.log.Error("failed to list transactions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load transactions"})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

func (h *Handler) holdingsSummary(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := parseInstrumentFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	summary, err := h.summary.HoldingsSummary(c.Request.Context(), scope, filter)
	if err != nil {
		h.log.Error("failed to compute holdings summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load holdings summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) dividendsSummary(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	summary, err := h.repo.DividendsSummary(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute dividends summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dividends summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) monthlyDividends(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	months, err := h.repo.MonthlyDividends(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute monthly dividends", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load monthly dividends"})
		return
	}
	c.JSON(http.StatusOK, months)
}

func (h *Handler) investedValueSeries(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	series, err := h.repo.InvestedValueSeries(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute invested value series", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load invested value series"})
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) marketValueSeries(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	series, err := h.summary.MarketValueSeries(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute market value series", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load market value series"})
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) sectorAllocations(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allocations, err := h.summary.SectorAllocations(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute sector allocations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sector allocations"})
		return
	}
	c.JSON(http.StatusOK, allocations)
}

func (h *Handler) allocations(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := parseInstrumentFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	groupBy, err := ParseGroupBy(c.DefaultQuery("groupBy", string(GroupBySector)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allocations, err := h.summary.Allocations(c.Request.Context(), scope, filter, groupBy)
	if err != nil {
		h.log.Error("failed to compute allocations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load allocations"})
		return
	}
	c.JSON(http.StatusOK, allocations)
}

func (h *Handler) allocationMatrix(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := parseInstrumentFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rowsBy, err := ParseGroupBy(c.Query("rows"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	colsBy, err := ParseGroupBy(c.Query("cols"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if rowsBy == colsBy {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rows and cols must be different dimensions"})
		return
	}
	matrix, err := h.summary.AllocationMatrix(c.Request.Context(), scope, filter, rowsBy, colsBy)
	if err != nil {
		h.log.Error("failed to compute allocation matrix", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load allocation matrix"})
		return
	}
	c.JSON(http.StatusOK, matrix)
}

func (h *Handler) allocationSeries(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := parseInstrumentFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	groupBy, err := ParseGroupBy(c.DefaultQuery("groupBy", string(GroupBySector)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	series, err := h.summary.AllocationSeries(c.Request.Context(), scope, filter, groupBy)
	if err != nil {
		h.log.Error("failed to compute allocation series", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load allocation series"})
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) dividendCalendar(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	calendar, err := h.summary.DividendCalendar(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute dividend calendar", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dividend calendar"})
		return
	}
	c.JSON(http.StatusOK, calendar)
}

func (h *Handler) passiveIncome(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := parseInstrumentFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	income, err := h.summary.PassiveIncome(c.Request.Context(), scope, filter)
	if err != nil {
		h.log.Error("failed to compute passive income", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load passive income"})
		return
	}
	c.JSON(http.StatusOK, income)
}

func (h *Handler) portfolioIRR(c *gin.Context) {
	scope, err := parseScope(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filter, err := parseInstrumentFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.summary.PortfolioIRR(c.Request.Context(), scope, filter)
	if err != nil {
		switch err {
		case xirr.ErrNoCashFlows, xirr.ErrNoSignChange:
			// Not enough activity yet for a rate of return to be defined
			// (e.g. no transactions, or nothing but buys and no current
			// market value) — a normal state, not a server error.
			c.JSON(http.StatusOK, gin.H{"irr": nil})
			return
		case xirr.ErrDidNotConverge:
			h.log.Warn("irr solver did not converge", zap.Error(err))
			c.JSON(http.StatusOK, gin.H{"irr": nil})
			return
		default:
			h.log.Error("failed to compute portfolio irr", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load portfolio irr"})
			return
		}
	}
	c.JSON(http.StatusOK, result)
}
