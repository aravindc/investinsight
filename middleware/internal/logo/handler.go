package logo

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	repo    *Repository
	service *Service
	log     *zap.Logger
}

func NewHandler(repo *Repository, service *Service, log *zap.Logger) *Handler {
	return &Handler{repo: repo, service: service, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/logos/backfill", h.backfill)
	router.GET("/logos/:instrumentId", h.getLogo)
}

func (h *Handler) backfill(c *gin.Context) {
	result, err := h.service.BackfillAll(c.Request.Context())
	if err != nil {
		h.log.Error("failed to backfill instrument logos", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to backfill instrument logos"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"instrumentsProcessed": result.Processed,
		"failed":               result.Failed,
	})
}

func (h *Handler) getLogo(c *gin.Context) {
	instrumentID, err := uuid.Parse(c.Param("instrumentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}

	logo, err := h.repo.LogoByInstrument(c.Request.Context(), instrumentID)
	if err != nil {
		h.log.Error("failed to load instrument logo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load instrument logo"})
		return
	}
	if logo == nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, logo.ContentType, logo.Data)
}
