package trading212

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
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
	router.GET("/trading212/connections", h.listConnections)
	router.GET("/trading212/cash", h.cash)
	router.PUT("/accounts/:id/trading212", h.connect)
	router.DELETE("/accounts/:id/trading212", h.disconnect)
	router.POST("/accounts/:id/trading212/sync", h.sync)
	router.GET("/accounts/:id/trading212/reconciliation", h.reconcile)
}

func (h *Handler) reconcile(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	rec, err := h.service.Reconcile(c.Request.Context(), accountID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, rec)
	case errors.Is(err, ErrNotConnected):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusBadGateway, gin.H{"error": "Trading 212 rejected the stored API key/secret; replace the key."})
	case errors.Is(err, ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Trading 212 rate limit hit; wait a few seconds and try again."})
	case errors.Is(err, ErrUnsupportedCurrency):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNoEncryptionKey):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server is missing CREDENTIALS_ENCRYPTION_KEY, so stored API keys can't be read."})
	default:
		h.log.Error("failed to reconcile with trading 212", zap.String("accountId", accountID.String()), zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "Couldn't reconcile with Trading 212."})
	}
}

func (h *Handler) listConnections(c *gin.Context) {
	conns, err := h.repo.List(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list trading 212 connections", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list Trading 212 connections"})
		return
	}
	if conns == nil {
		conns = []Connection{}
	}
	c.JSON(http.StatusOK, conns)
}

type connectRequest struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
}

func (h *Handler) connect(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}

	var req connectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	creds := Credentials{APIKey: strings.TrimSpace(req.APIKey), APISecret: strings.TrimSpace(req.APISecret)}
	if creds.APIKey == "" || creds.APISecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "apiKey and apiSecret are required"})
		return
	}

	conn, err := h.service.Connect(c.Request.Context(), accountID, creds)
	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.Is(err, ErrNoEncryptionKey):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server is missing CREDENTIALS_ENCRYPTION_KEY, so API keys can't be stored."})
		case errors.Is(err, ErrUnauthorized):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Trading 212 rejected this API key/secret. Check both values and that the key has account, history and export permissions."})
		case errors.Is(err, ErrUnsupportedCurrency):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, ErrRateLimited):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Trading 212 rate limit hit; wait a few seconds and try again."})
		case errors.As(err, &pqErr) && pqErr.Constraint == "trading212_connections_account_id_fkey":
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		default:
			h.log.Error("failed to connect trading 212", zap.String("accountId", accountID.String()), zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "Couldn't verify the credentials with Trading 212."})
		}
		return
	}
	h.log.Info("trading 212 connected", zap.String("accountId", accountID.String()))
	c.JSON(http.StatusOK, conn)
}

func (h *Handler) disconnect(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	if err := h.repo.Delete(c.Request.Context(), accountID); err != nil {
		h.log.Error("failed to remove trading 212 connection", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove Trading 212 connection"})
		return
	}
	h.log.Info("trading 212 disconnected", zap.String("accountId", accountID.String()))
	c.Status(http.StatusNoContent)
}

func (h *Handler) sync(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	err := h.service.StartSync(c.Request.Context(), accountID)
	switch {
	case err == nil:
		c.JSON(http.StatusAccepted, gin.H{"status": SyncRunning})
	case errors.Is(err, ErrNotConnected):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrSyncRunning):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNoEncryptionKey):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Server is missing CREDENTIALS_ENCRYPTION_KEY, so stored API keys can't be read."})
	default:
		h.log.Error("failed to start trading 212 sync", zap.String("accountId", accountID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start sync"})
	}
}

func (h *Handler) cash(c *gin.Context) {
	scope, err := ParseCashScope(c.Query("scope"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	summary, err := h.service.Cash(c.Request.Context(), scope)
	if err != nil {
		h.log.Error("failed to compute trading 212 cash", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load cash balances"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func parseAccountID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return uuid.Nil, false
	}
	return id, true
}
