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

		// TODO: Add other repositories as they are implemented
		// fx.Annotate(
		//     repository.NewProductRepository,
		//     fx.As(new(repository.ProductRepository)),
		// ),
		// fx.Annotate(
		//     repository.NewOrderRepository,
		//     fx.As(new(repository.OrderRepository)),
		// ),
		// fx.Annotate(
		//     repository.NewInventoryRepository,
		//     fx.As(new(repository.InventoryRepository)),
		// ),
		// fx.Annotate(
		//     repository.NewPaymentRepository,
		//     fx.As(new(repository.PaymentRepository)),
		// ),
		// fx.Annotate(
		//     repository.NewNotificationRepository,
		//     fx.As(new(repository.NotificationRepository)),
		// ),
		// fx.Annotate(
		//     repository.NewAuditLogRepository,
		//     fx.As(new(repository.AuditLogRepository)),
		// ),
	),
)
