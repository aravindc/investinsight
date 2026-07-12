package account

import (
	"net/http"
	"strings"

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
	router.GET("/accounts", h.listAccounts)
	router.POST("/accounts", h.createAccount)
}

func (h *Handler) listAccounts(c *gin.Context) {
	accounts, err := h.repo.List(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list accounts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list accounts"})
		return
	}
	if accounts == nil {
		accounts = []Account{}
	}
	c.JSON(http.StatusOK, accounts)
}

type createAccountRequest struct {
	HouseholdID string `json:"householdId"`
	Name        string `json:"name"`
}

func (h *Handler) createAccount(c *gin.Context) {
	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	householdID, err := uuid.Parse(req.HouseholdID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing householdId"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	created, err := h.repo.Create(c.Request.Context(), householdID, name)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Constraint {
			case "accounts_household_id_name_key":
				c.JSON(http.StatusConflict, gin.H{"error": "an account with this name already exists in this household"})
				return
			case "accounts_household_id_fkey":
				c.JSON(http.StatusBadRequest, gin.H{"error": "household not found"})
				return
			}
		}
		h.log.Error("failed to create account", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}
	c.JSON(http.StatusCreated, created)
}
