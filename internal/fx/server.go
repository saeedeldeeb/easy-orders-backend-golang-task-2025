package fx

import (
	"context"
	middleware2 "easy-orders-backend/internal/api/middleware"
	"fmt"
	"net/http"
	"time"

	"easy-orders-backend/internal/api/handlers"
	"easy-orders-backend/internal/api/routes"
	"easy-orders-backend/internal/config"
	"easy-orders-backend/pkg/logger"

	_ "easy-orders-backend/docs" // Import generated Swagger docs

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
)

// ServerModule provides HTTP server
var ServerModule = fx.Module("server",
	fx.Provide(NewGinEngine),
	fx.Provide(NewHTTPServer),
	fx.Invoke(RegisterServerLifecycle),
)

// NewGinEngine creates a new Gin engine
func NewGinEngine(
	cfg *config.Config,
	logger *logger.Logger,
	errorMiddleware *middleware2.ErrorMiddleware,
	authMiddleware *middleware2.AuthMiddleware,
	corsMiddleware *middleware2.CORSMiddleware,
	rateLimiter *middleware2.RateLimiter,
	validationMiddleware *middleware2.ValidationMiddleware,
	userHandler *handlers.UserHandler,
	productHandler *handlers.ProductHandler,
	orderHandler *handlers.OrderHandler,
	paymentHandler *handlers.PaymentHandler,
	inventoryHandler *handlers.InventoryHandler,
	adminHandler *handlers.AdminHandler,
) *gin.Engine {
	// Set gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()

	// Add core middleware in order
	engine.Use(gin.Recovery())            // Panic recovery
	engine.Use(corsMiddleware.Handler())  // CORS handling
	engine.Use(rateLimiter.Limit())       // Rate limiting
	engine.Use(errorMiddleware.Handler()) // Centralized error handling

	// Add basic request logging
	engine.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Info("HTTP Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start),
			"client_ip", c.ClientIP(),
		)
	})

	// Health check endpoint
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"service":   "easy-orders-backend",
		})
	})

	// Swagger documentation endpoint
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 group
	v1 := engine.Group("/api/v1")
	{
		// Public routes (no authentication required)
		routes.RegisterUserRoutes(v1, userHandler, validationMiddleware) // Includes auth endpoint

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			routes.RegisterProductRoutes(protected, productHandler, inventoryHandler, validationMiddleware)
			routes.RegisterOrderRoutes(protected, orderHandler, validationMiddleware)
			routes.RegisterPaymentRoutes(protected, paymentHandler, validationMiddleware)
		}

		// Admin routes (require admin role)
		admin := v1.Group("")
		admin.Use(authMiddleware.RequireAuth())
		admin.Use(authMiddleware.RequireAdmin())
		{
			routes.RegisterAdminRoutes(admin, adminHandler, inventoryHandler, validationMiddleware)
		}

		// Health check under API version
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	return engine
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(cfg *config.Config, engine *gin.Engine) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
}

// RegisterServerLifecycle registers server start/stop hooks
func RegisterServerLifecycle(lc fx.Lifecycle, server *http.Server, logger *logger.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting HTTP server", "addr", server.Addr)

			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server failed to start", "error", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server...")

			// Create a context with timeout for graceful shutdown
			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			return server.Shutdown(shutdownCtx)
		},
	})
}
