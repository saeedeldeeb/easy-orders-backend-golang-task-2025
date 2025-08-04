package fx

import (
	"context"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/logger"

	"go.uber.org/fx"
)

// ReportsModule provides enhanced report generation dependencies
var ReportsModule = fx.Module("reports",
	fx.Provide(
		// Enhanced report service
		services.NewEnhancedReportService,
	),

	// Lifecycle hooks
	fx.Invoke(func(lc fx.Lifecycle, enhancedService *services.EnhancedReportService, logger *logger.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("Enhanced report system initialized")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Enhanced report system shutting down")
				return nil
			},
		})
	}),
)
