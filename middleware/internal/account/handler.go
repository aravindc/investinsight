package account

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
	router.GET("/accounts", h.listAccounts)
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
