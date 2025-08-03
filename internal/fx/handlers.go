package fx

import (
	"easy-orders-backend/internal/api/handlers"

	"go.uber.org/fx"
)

// HandlersModule provides all HTTP handlers
var HandlersModule = fx.Module("handlers",
	fx.Provide(
		handlers.NewUserHandler,

		// TODO: Add other handlers as they are implemented
		// handlers.NewProductHandler,
		// handlers.NewOrderHandler,
		// handlers.NewInventoryHandler,
		// handlers.NewPaymentHandler,
		// handlers.NewNotificationHandler,
		// handlers.NewReportHandler,
	),
)
