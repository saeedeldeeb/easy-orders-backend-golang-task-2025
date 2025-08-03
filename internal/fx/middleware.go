package fx

import (
	"easy-orders-backend/internal/config"
	"easy-orders-backend/internal/middleware"
	"easy-orders-backend/pkg/jwt"
	"easy-orders-backend/pkg/logger"

	"go.uber.org/fx"
)

// MiddlewareModule provides all middleware dependencies
var MiddlewareModule = fx.Module("middleware",
	fx.Provide(
		// JWT Token Manager
		func(cfg *config.Config) *jwt.TokenManager {
			return jwt.NewTokenManager(&cfg.JWT)
		},

		// Auth Middleware
		middleware.NewAuthMiddleware,

		// CORS Middleware
		middleware.NewCORSMiddleware,

		// Rate Limiting Middleware
		func(logger *logger.Logger) *middleware.RateLimiter {
			return middleware.CreateStandardLimiter(logger)
		},

		// Validation Middleware
		middleware.NewValidationMiddleware,

		// Additional rate limiters for different use cases
		fx.Annotate(
			func(logger *logger.Logger) *middleware.RateLimiter {
				return middleware.CreateStrictLimiter(logger)
			},
			fx.ResultTags(`name:"strict_limiter"`),
		),

		fx.Annotate(
			func(logger *logger.Logger) *middleware.RateLimiter {
				return middleware.CreateGenerousLimiter(logger)
			},
			fx.ResultTags(`name:"generous_limiter"`),
		),
	),
)
