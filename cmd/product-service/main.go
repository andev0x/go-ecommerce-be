package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"ecommerce/internal/product/config"
	"ecommerce/internal/product/handler"
	"ecommerce/internal/product/repository"
	"ecommerce/internal/product/service"
	"ecommerce/pkg/database"
	"ecommerce/pkg/logger"
	"ecommerce/pkg/redis"
)

func main() {
	// Initialize logger
	logger := logger.NewLogger()
	
	// Load configuration
	cfg := config.Load()
	
	// Initialize database
	db, err := database.NewPostgresConnection(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", err)
	}
	defer database.Close(db)
	
	// Initialize Redis
	redisClient, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", err)
	}
	defer redisClient.Close()
	
	// Initialize repository
	repo := repository.NewProductRepository(db, redisClient, logger)
	
	// Initialize service
	productService := service.NewProductService(repo, logger)
	
	// Initialize handlers
	httpHandler := handler.NewHTTPHandler(productService, logger)
	
	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Register HTTP routes
	httpHandler.RegisterRoutes(router)
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTP.Port),
		Handler: router,
	}
	
	// Start HTTP server
	go func() {
		logger.Info(fmt.Sprintf("HTTP server listening on port %s", cfg.HTTP.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", err)
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down servers...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", err)
	}
	
	logger.Info("Server exited")
}