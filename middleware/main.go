package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"investinsight/middleware/internal/config"
	"investinsight/middleware/internal/db"
	"investinsight/middleware/internal/ingest"
	appLogger "investinsight/middleware/internal/logger"
	"investinsight/middleware/internal/marketprice"
	"investinsight/middleware/internal/migrate"
	"investinsight/middleware/internal/portfolio"
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
		AllowOrigins: []string{"http://localhost:5173"},
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

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
