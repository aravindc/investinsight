package marketprice

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/market-prices/backfill", h.backfill)
}

func (h *Handler) backfill(c *gin.Context) {
	result, err := h.service.BackfillAll(c.Request.Context())
	if err != nil {
		h.log.Error("failed to backfill market prices", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to backfill market prices"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"instrumentsProcessed": result.Processed,
		"failed":               result.Failed,
	})
}
