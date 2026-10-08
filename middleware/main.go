package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"investinsight/middleware/internal/account"
	"investinsight/middleware/internal/cashisa"
	"investinsight/middleware/internal/config"
	"investinsight/middleware/internal/db"
	"investinsight/middleware/internal/household"
	"investinsight/middleware/internal/ingest"
	"investinsight/middleware/internal/instrument"
	appLogger "investinsight/middleware/internal/logger"
	"investinsight/middleware/internal/logo"
	"investinsight/middleware/internal/marketprice"
	"investinsight/middleware/internal/migrate"
	"investinsight/middleware/internal/portfolio"
	"investinsight/middleware/internal/trading212"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := appLogger.New(false)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	database, err := db.Connect(cfg.ConnString())
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close()
	logger.Info("connected to database", zap.String("host", cfg.DBHost))

	if err := migrate.Up(database); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}
	logger.Info("database migrations up to date")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
	}))
	router.Use(requestLogger(logger))

	api := router.Group("/api")

	marketPriceRepo := marketprice.NewRepository(database)
	yahooClient := marketprice.NewYahooClient()
	marketPriceService := marketprice.NewService(marketPriceRepo, yahooClient, logger)
	marketPriceHandler := marketprice.NewHandler(marketPriceService, logger)
	marketPriceHandler.RegisterRoutes(api)

	portfolioRepo := portfolio.NewRepository(database)
	summaryService := portfolio.NewSummaryService(portfolioRepo, marketPriceService, logger)
	portfolioHandler := portfolio.NewHandler(portfolioRepo, summaryService, logger)
	portfolioHandler.RegisterRoutes(api)

	ingestEngine := ingest.NewEngine(database)
	ingestHandler := ingest.NewHandler(ingestEngine, logger)
	ingestHandler.RegisterRoutes(api)

	accountRepo := account.NewRepository(database)
	accountHandler := account.NewHandler(accountRepo, logger)
	accountHandler.RegisterRoutes(api)

	householdRepo := household.NewRepository(database)
	householdHandler := household.NewHandler(householdRepo, logger)
	householdHandler.RegisterRoutes(api)

	logoRepo := logo.NewRepository(database)
	logoClient := logo.NewClient(cfg.LogoDevToken)
	logoService := logo.NewService(logoRepo, logoClient, logger)
	logoHandler := logo.NewHandler(logoRepo, logoService, logger)
	logoHandler.RegisterRoutes(api)

	credentialsCipher, err := trading212.NewCipher(cfg.CredentialsEncryptionKey)
	if err != nil {
		logger.Fatal("invalid credentials encryption key", zap.Error(err))
	}
	if credentialsCipher == nil {
		logger.Warn("CREDENTIALS_ENCRYPTION_KEY not set; Trading 212 connections are disabled")
	}
	t212Client, err := trading212.NewClient(cfg.Trading212BaseURL)
	if err != nil {
		logger.Fatal("invalid Trading 212 base URL", zap.Error(err))
	}
	t212Repo := trading212.NewRepository(database)
	if err := t212Repo.FailInterruptedSyncs(context.Background()); err != nil {
		logger.Error("failed to reset interrupted Trading 212 syncs", zap.Error(err))
	}
	t212Service := trading212.NewService(t212Repo, t212Client, credentialsCipher, ingestEngine, summaryService, logger)
	t212Handler := trading212.NewHandler(t212Repo, t212Service, logger)
	t212Handler.RegisterRoutes(api)

	cashISARepo := cashisa.NewRepository(database)
	cashISAHandler := cashisa.NewHandler(cashISARepo, logger)
	cashISAHandler.RegisterRoutes(api)

	instrumentRepo := instrument.NewRepository(database)
	instrumentHandler := instrument.NewHandler(instrumentRepo, logger)
	instrumentHandler.RegisterRoutes(api)

	// Liveness + readiness: reports unhealthy if the database is unreachable,
	// so Docker's healthcheck reflects whether the API can actually serve.
	router.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := database.PingContext(ctx); err != nil {
			logger.Warn("health check: database unreachable", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "ok"})
	})

	addr := ":" + cfg.AppPort
	logger.Info("starting server", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		logger.Fatal("server stopped", zap.Error(err))
	}
}

func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	}
}
