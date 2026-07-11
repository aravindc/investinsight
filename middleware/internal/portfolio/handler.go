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
	router.GET("/portfolio/irr", h.portfolioIRR)
	router.GET("/portfolio/passive-income", h.passiveIncome)
}

func (h *Handler) listTransactions(c *gin.Context) {
	transactions, err := h.repo.ListTransactions(c.Request.Context(), 200)
	if err != nil {
		h.log.Error("failed to list transactions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load transactions"})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

func (h *Handler) holdingsSummary(c *gin.Context) {
	summary, err := h.summary.HoldingsSummary(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute holdings summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load holdings summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) dividendsSummary(c *gin.Context) {
	summary, err := h.repo.DividendsSummary(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute dividends summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dividends summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) monthlyDividends(c *gin.Context) {
	months, err := h.repo.MonthlyDividends(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute monthly dividends", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load monthly dividends"})
		return
	}
	c.JSON(http.StatusOK, months)
}

func (h *Handler) investedValueSeries(c *gin.Context) {
	series, err := h.repo.InvestedValueSeries(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute invested value series", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load invested value series"})
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) marketValueSeries(c *gin.Context) {
	series, err := h.summary.MarketValueSeries(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute market value series", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load market value series"})
		return
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) sectorAllocations(c *gin.Context) {
	allocations, err := h.summary.SectorAllocations(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute sector allocations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load sector allocations"})
		return
	}
	c.JSON(http.StatusOK, allocations)
}

func (h *Handler) passiveIncome(c *gin.Context) {
	income, err := h.summary.PassiveIncome(c.Request.Context())
	if err != nil {
		h.log.Error("failed to compute passive income", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load passive income"})
		return
	}
	c.JSON(http.StatusOK, income)
}

func (h *Handler) portfolioIRR(c *gin.Context) {
	result, err := h.summary.PortfolioIRR(c.Request.Context())
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
