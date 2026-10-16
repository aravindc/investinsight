package logo

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

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
	// refresh=true re-fetches every logo, not just missing ones.
	result, err := h.service.BackfillAll(c.Request.Context(), c.Query("refresh") == "true")
	if err != nil {
		h.log.Error("failed to backfill instrument logos", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to backfill instrument logos"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"instrumentsProcessed": result.Processed,
		"failed":               result.Failed,
		"removed":              result.Removed,
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
	// Browsers revalidate on every use (no-cache) against an ETag of the
	// image, so a logo replaced or removed by a refresh shows at once, while
	// an unchanged one costs only a 304. A long max-age kept stale
	// placeholders on screen for a day after a refresh.
	c.Header("Cache-Control", "no-cache")
	if logo == nil {
		c.Status(http.StatusNotFound)
		return
	}

	etag := logoETag(logo.Data)
	c.Header("ETag", etag)
	if etagMatches(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, logo.ContentType, logo.Data)
}

// logoETag is a strong ETag from the image bytes.
func logoETag(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

// etagMatches reports whether an If-None-Match header names etag (or *).
func etagMatches(ifNoneMatch, etag string) bool {
	for _, candidate := range strings.Split(ifNoneMatch, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(candidate), "W/")
		if candidate == etag || candidate == "*" {
			return true
		}
	}
	return false
}
