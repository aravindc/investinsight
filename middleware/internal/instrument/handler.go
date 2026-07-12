package instrument

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"investinsight/middleware/internal/sector"
)

type Handler struct {
	repo *Repository
	log  *zap.Logger
}

func NewHandler(repo *Repository, log *zap.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/instruments", h.listInstruments)
	router.PATCH("/instruments/:id/sector", h.updateSector)
}

func (h *Handler) listInstruments(c *gin.Context) {
	instruments, err := h.repo.List(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list instruments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list instruments"})
		return
	}
	if instruments == nil {
		instruments = []Instrument{}
	}
	c.JSON(http.StatusOK, instruments)
}

type updateSectorRequest struct {
	Sector string `json:"sector"`
}

func (h *Handler) updateSector(c *gin.Context) {
	instrumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}

	var req updateSectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if !sector.IsValid(req.Sector) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unrecognized sector"})
		return
	}

	if err := h.repo.UpdateSector(c.Request.Context(), instrumentID, req.Sector); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "instrument not found"})
			return
		}
		h.log.Error("failed to update instrument sector", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instrument sector"})
		return
	}
	c.Status(http.StatusNoContent)
}
