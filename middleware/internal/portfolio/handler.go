package portfolio

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	repo *Repository
	log  *zap.Logger
}

func NewHandler(repo *Repository, log *zap.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/transactions", h.listTransactions)
	router.GET("/portfolio/holdings-summary", h.holdingsSummary)
	router.GET("/portfolio/dividends-summary", h.dividendsSummary)
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
	summary, err := h.repo.HoldingsSummary(c.Request.Context())
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
