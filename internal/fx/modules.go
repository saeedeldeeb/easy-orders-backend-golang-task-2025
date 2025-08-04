package fx

import (
	"context"

	"easy-orders-backend/internal/config"
	migrations "easy-orders-backend/internal/database"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ConfigModule provides application configuration
var ConfigModule = fx.Module("config",
	fx.Provide(config.Load),
)

// LoggerModule provides structured logging
var LoggerModule = fx.Module("logger",
	fx.Provide(
		func(cfg *config.Config) (*logger.Logger, error) {
			if cfg.Server.Environment == "development" {
				return logger.NewDevelopment()
			}
			return logger.New(cfg.Server.LogLevel)
		},
	),
	fx.Invoke(func(logger *logger.Logger) {
		// Replace global zap logger
		zap.ReplaceGlobals(logger.SugaredLogger.Desugar())
	}),
)

// DatabaseModule provides database connection
var DatabaseModule = fx.Module("database",
	fx.Provide(
		func(cfg *config.Config, logger *logger.Logger) (*database.DB, error) {
			logger.Info("Connecting to database...")
			db, err := database.New(&cfg.Database)
			if err != nil {
				logger.Error("Failed to connect to database", "error", err)
				return nil, err
			}
			logger.Info("Database connection established")
			return db, nil
		},
	),
	// Provide the underlying *gorm.DB for migrator
	fx.Provide(func(db *database.DB) *gorm.DB {
		return db.DB
	}),
	fx.Provide(migrations.NewMigrator),
	fx.Invoke(func(lc fx.Lifecycle, db *database.DB, migrator *migrations.Migrator, logger *logger.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("Running database migrations...")
				return migrator.RunMigrations()
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Closing database connection...")
				return db.Close()
			},
		})
	}),
)

// CoreModules combines all core application modules
var CoreModules = fx.Options(
	ConfigModule,
	LoggerModule,
	DatabaseModule,
)

// ApplicationModules combines all application-specific modules
var ApplicationModules = fx.Options(
	RepositoriesModule,
	ServicesModule,
	HandlersModule,
	ConcurrencyModule,
	WorkersModule,
)
