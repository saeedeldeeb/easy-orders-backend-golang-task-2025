package fx

import (
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/workers"

	"go.uber.org/fx"
)

// WorkersModule provides worker pool and background service dependencies
var WorkersModule = fx.Module("workers",
	fx.Provide(
		// Pool manager
		workers.NewPoolManager,

		// Background service
		services.NewBackgroundService,
	),
)
