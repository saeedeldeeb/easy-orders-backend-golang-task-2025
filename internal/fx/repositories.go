package fx

import (
	"easy-orders-backend/internal/repository"

	"go.uber.org/fx"
)

// RepositoriesModule provides all data access repositories
var RepositoriesModule = fx.Module("repositories",
	fx.Provide(
		// User repository
		fx.Annotate(
			repository.NewUserRepository,
			fx.As(new(repository.UserRepository)),
		),

		// Product repository
		fx.Annotate(
			repository.NewProductRepository,
			fx.As(new(repository.ProductRepository)),
		),

		// Order repository
		fx.Annotate(
			repository.NewOrderRepository,
			fx.As(new(repository.OrderRepository)),
		),

		// Order item repository
		fx.Annotate(
			repository.NewOrderItemRepository,
			fx.As(new(repository.OrderItemRepository)),
		),

		// Inventory repository
		fx.Annotate(
			repository.NewInventoryRepository,
			fx.As(new(repository.InventoryRepository)),
		),

		// Payment repository
		fx.Annotate(
			repository.NewPaymentRepository,
			fx.As(new(repository.PaymentRepository)),
		),

		// Notification repository
		fx.Annotate(
			repository.NewNotificationRepository,
			fx.As(new(repository.NotificationRepository)),
		),

		// Audit log repository
		fx.Annotate(
			repository.NewAuditLogRepository,
			fx.As(new(repository.AuditLogRepository)),
		),
	),
)
