package fx

import (
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/notifications"

	"go.uber.org/fx"
)

// NotificationsModule provides notification system dependencies
var NotificationsModule = fx.Module("notifications",
	fx.Provide(
		// Core notification components
		notifications.NewNotificationProvider,
		notifications.NewTemplateManager,
		func(config *notifications.DispatcherConfig, provider *notifications.NotificationProvider, logger *logger.Logger) *notifications.NotificationDispatcher {
			if config == nil {
				config = notifications.DefaultDispatcherConfig()
			}
			return notifications.NewNotificationDispatcher(config, provider, logger)
		},

		// Enhanced notification service
		services.NewEnhancedNotificationService,

		// Default dispatcher config
		func() *notifications.DispatcherConfig {
			return notifications.DefaultDispatcherConfig()
		},
	),
)
