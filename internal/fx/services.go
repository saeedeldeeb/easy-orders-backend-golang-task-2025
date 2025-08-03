package fx

import (
	"easy-orders-backend/internal/services"

	"go.uber.org/fx"
)

// ServicesModule provides all business logic services
var ServicesModule = fx.Module("services",
	fx.Provide(
		// User service
		fx.Annotate(
			services.NewUserService,
			fx.As(new(services.UserService)),
		),

		// TODO: Add other services as they are implemented
		// fx.Annotate(
		//     services.NewProductService,
		//     fx.As(new(services.ProductService)),
		// ),
		// fx.Annotate(
		//     services.NewOrderService,
		//     fx.As(new(services.OrderService)),
		// ),
		// fx.Annotate(
		//     services.NewInventoryService,
		//     fx.As(new(services.InventoryService)),
		// ),
		// fx.Annotate(
		//     services.NewPaymentService,
		//     fx.As(new(services.PaymentService)),
		// ),
		// fx.Annotate(
		//     services.NewNotificationService,
		//     fx.As(new(services.NotificationService)),
		// ),
		// fx.Annotate(
		//     services.NewReportService,
		//     fx.As(new(services.ReportService)),
		// ),
	),
)
