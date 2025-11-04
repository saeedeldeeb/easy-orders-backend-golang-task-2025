package fx

import (
	middleware2 "easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/config"
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

		// Error Middleware
		middleware2.NewErrorMiddleware,

		// Auth Middleware
		middleware2.NewAuthMiddleware,

		// CORS Middleware
		middleware2.NewCORSMiddleware,

		// Rate Limiting Middleware
		func(logger *logger.Logger) *middleware2.RateLimiter {
			return middleware2.CreateStandardLimiter(logger)
		},

		// Validation Middleware
		middleware2.NewValidationMiddleware,

		// Additional rate limiters for different use cases
		fx.Annotate(
			func(logger *logger.Logger) *middleware2.RateLimiter {
				return middleware2.CreateStrictLimiter(logger)
			},
			fx.ResultTags(`name:"strict_limiter"`),
		),

		fx.Annotate(
			func(logger *logger.Logger) *middleware2.RateLimiter {
				return middleware2.CreateGenerousLimiter(logger)
			},
			fx.ResultTags(`name:"generous_limiter"`),
		),
	),
)
