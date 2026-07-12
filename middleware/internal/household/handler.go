package household

import (
	"net/http"
	"strings"

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
	router.GET("/households", h.listHouseholds)
	router.POST("/households", h.createHousehold)
}

func (h *Handler) listHouseholds(c *gin.Context) {
	households, err := h.repo.List(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list households", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list households"})
		return
	}
	if households == nil {
		households = []Household{}
	}
	c.JSON(http.StatusOK, households)
}

type createHouseholdRequest struct {
	Name string `json:"name"`
}

func (h *Handler) createHousehold(c *gin.Context) {
	var req createHouseholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	created, err := h.repo.Create(c.Request.Context(), name)
	if err != nil {
		h.log.Error("failed to create household", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create household"})
		return
	}
	c.JSON(http.StatusCreated, created)
}
