package fx

import (
	"easy-orders-backend/internal/api/handlers"

	"go.uber.org/fx"
)

// HandlersModule provides all HTTP handlers
var HandlersModule = fx.Module("handlers",
	fx.Provide(
		handlers.NewUserHandler,
		handlers.NewProductHandler,
		handlers.NewOrderHandler,
		handlers.NewPaymentHandler,
		handlers.NewInventoryHandler,
		handlers.NewAdminHandler,
	),
)
