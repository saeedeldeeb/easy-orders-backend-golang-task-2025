package fx

import (
	"context"
	"time"

	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/payments"

	"go.uber.org/fx"
)

// PaymentsModule provides payment processing infrastructure dependencies
var PaymentsModule = fx.Module("payments",
	fx.Provide(
		// Payment gateway manager
		payments.NewPaymentGatewayManager,

		// Idempotency manager
		func(logger *logger.Logger) *payments.IdempotencyManager {
			return payments.NewIdempotencyManager(24*time.Hour, logger) // 24 hour TTL
		},

		// Circuit breaker manager
		payments.NewCircuitBreakerManager,
	),

	// Decorate the gateway manager to register mock gateways
	fx.Decorate(func(gatewayManager *payments.PaymentGatewayManager, logger *logger.Logger) *payments.PaymentGatewayManager {
		// Register mock gateways for testing
		mockStripe := payments.NewMockPaymentGateway(payments.GatewayTypeStripe, 0.05, 500*time.Millisecond, logger)
		mockPayPal := payments.NewMockPaymentGateway(payments.GatewayTypePayPal, 0.03, 300*time.Millisecond, logger)
		mockSquare := payments.NewMockPaymentGateway(payments.GatewayTypeSquare, 0.04, 400*time.Millisecond, logger)

		gatewayManager.RegisterGateway(mockStripe)
		gatewayManager.RegisterGateway(mockPayPal)
		gatewayManager.RegisterGateway(mockSquare)

		logger.Info("Payment gateways registered", "count", 3)
		return gatewayManager
	}),

	// Lifecycle hooks
	fx.Invoke(func(lc fx.Lifecycle, idempotencyManager *payments.IdempotencyManager, logger *logger.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("Enhanced payment system initialized")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Shutting down enhanced payment system")
				idempotencyManager.Stop()
				return nil
			},
		})
	}),
)
