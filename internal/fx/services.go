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

		// Product service
		fx.Annotate(
			services.NewProductService,
			fx.As(new(services.ProductService)),
		),

		// Inventory service
		fx.Annotate(
			services.NewInventoryService,
			fx.As(new(services.InventoryService)),
		),

		// Order service
		fx.Annotate(
			services.NewOrderService,
			fx.As(new(services.OrderService)),
		),

		// Payment service
		fx.Annotate(
			services.NewPaymentService,
			fx.As(new(services.PaymentService)),
		),

		// Notification service
		fx.Annotate(
			services.NewNotificationService,
			fx.As(new(services.NotificationService)),
		),

		// Report service
		fx.Annotate(
			services.NewReportService,
			fx.As(new(services.ReportService)),
		),
	),
)
