package instrument

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
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
	router.GET("/instruments", h.listInstruments)
	router.PUT("/instruments/:id/classification", h.updateClassification)
	router.POST("/instruments/classification-import", h.importClassifications)
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

func (h *Handler) updateClassification(c *gin.Context) {
	instrumentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}

	var req Classification
	if err := c.ShouldBindJSON(&req); err != nil || req.Category == "" || req.SubCategory == "" || req.Sector == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category, subCategory and sector are required"})
		return
	}

	// Valid values live in the categories/sub_categories/sectors tables; the
	// foreign keys reject anything else.
	if err := h.repo.UpdateClassification(c.Request.Context(), instrumentID, req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "instrument not found"})
			return
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Constraint {
			case "instruments_sector_fkey":
				c.JSON(http.StatusBadRequest, gin.H{"error": "unrecognized sector"})
				return
			case "instruments_sub_category_fkey":
				c.JSON(http.StatusBadRequest, gin.H{"error": "unrecognized category / sub-category combination"})
				return
			case "instruments_sub_sector_fkey":
				c.JSON(http.StatusBadRequest, gin.H{"error": "unrecognized sector / sub-sector combination"})
				return
			}
		}
		h.log.Error("failed to update instrument classification", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update instrument classification"})
		return
	}
	c.Status(http.StatusNoContent)
}

// maxImportBytes bounds an uploaded classification CSV.
const maxImportBytes = 5 << 20

func (h *Handler) importClassifications(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing \"file\" form field"})
		return
	}
	if fileHeader.Size > maxImportBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is larger than 5 MB"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "couldn't read the uploaded file"})
		return
	}
	defer file.Close()

	rows, issues, err := ParseClassificationCSV(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.repo.ImportClassifications(c.Request.Context(), rows, issues)
	if err != nil {
		h.log.Error("classification import failed", zap.String("filename", fileHeader.Filename), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import failed; nothing was changed"})
		return
	}
	h.log.Info("classification import applied",
		zap.String("filename", fileHeader.Filename),
		zap.Int("updated", result.Updated), zap.Int("unchanged", result.Unchanged), zap.Int("skipped", len(result.Skipped)))
	c.JSON(http.StatusOK, result)
}
