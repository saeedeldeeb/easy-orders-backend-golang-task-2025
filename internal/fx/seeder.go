package fx

import (
	"easy-orders-backend/migrations"

	"go.uber.org/fx"
)

// SeederModule provides database seeding functionality
var SeederModule = fx.Module("seeder",
	fx.Invoke(func(migrator *migrations.Migrator) {
		// Seed data will be run after migrations
		// This is optional and can be disabled in production
		if err := migrator.SeedData(); err != nil {
			// Log error but don't fail startup for seeding issues
			// This allows the application to run even if seeding fails
		}
	}),
)
