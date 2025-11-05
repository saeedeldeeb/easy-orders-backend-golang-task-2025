package fx

import (
	"easy-orders-backend/pkg/concurrency"

	"go.uber.org/fx"
)

// ConcurrencyModule provides concurrency-related dependencies
var ConcurrencyModule = fx.Module("concurrency",
	fx.Provide(
		// Distributed lock implementation
		fx.Annotate(
			concurrency.NewRedisLock,
			fx.As(new(concurrency.DistributedLock)),
		),

		// Lock manager
		concurrency.NewLockManager,
	),
)
