package ingest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	engine *Engine
	log    *zap.Logger
}

func NewHandler(engine *Engine, log *zap.Logger) *Handler {
	return &Handler{engine: engine, log: log}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/csv-upload", h.uploadCSV)
}

type uploadResponse struct {
	Inserted  int      `json:"inserted"`
	Duplicate int      `json:"duplicate"`
	Errors    []string `json:"errors"`
}

func (h *Handler) uploadCSV(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing \"file\" form field"})
		return
	}

	hasHeader := c.DefaultPostForm("hasHeader", "true") != "false"

	file, err := fileHeader.Open()
	if err != nil {
		h.log.Error("failed to open uploaded file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
		return
	}
	defer file.Close()

	result, err := h.engine.LoadCSV(c.Request.Context(), file, hasHeader)
	if err != nil {
		h.log.Error("failed to load CSV", zap.Error(err), zap.String("filename", fileHeader.Filename))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CSV"})
		return
	}

	errMessages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		errMessages = append(errMessages, e.Error())
	}

	h.log.Info("CSV upload processed",
		zap.String("filename", fileHeader.Filename),
		zap.Int("inserted", result.Inserted),
		zap.Int("duplicate", result.Duplicate),
		zap.Int("errors", len(result.Errors)),
	)

	c.JSON(http.StatusOK, uploadResponse{
		Inserted:  result.Inserted,
		Duplicate: result.Duplicate,
		Errors:    errMessages,
	})
}
