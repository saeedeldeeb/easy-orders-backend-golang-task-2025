package fx

import (
	"context"
	"easy-orders-backend/pkg/logger"

	"go.uber.org/fx"
)

// ReportsModule provides report system lifecycle hooks
var ReportsModule = fx.Module("reports",
	// Lifecycle hooks
	fx.Invoke(func(lc fx.Lifecycle, logger *logger.Logger) {
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
