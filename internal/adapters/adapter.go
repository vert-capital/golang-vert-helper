package adapters

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// ========== Application Initializer ==========

// ApplicationInitializer handles initialization of the entire application
type ApplicationInitializer struct {
	db    *gorm.DB
	repos *RepositoryFactory
}

// NewApplicationInitializer creates a new application initializer
func NewApplicationInitializer(config *Config) (*ApplicationInitializer, error) {
	// Create database adapter
	dsn := config.GetPostgresDSN()
	dbAdapter, err := NewDatabaseAdapter(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create database adapter: %w", err)
	}

	// Run auto migrations
	if err := dbAdapter.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create repository factory
	repos := NewRepositoryFactory(dbAdapter.DB)

	return &ApplicationInitializer{
		db:    dbAdapter.DB,
		repos: repos,
	}, nil
}

// GetRepositoryFactory returns the repository factory
func (ai *ApplicationInitializer) GetRepositoryFactory() *RepositoryFactory {
	return ai.repos
}

// GetDatabase returns the GORM database connection
func (ai *ApplicationInitializer) GetDatabase() *gorm.DB {
	return ai.db
}

// Close closes the database connection
func (ai *ApplicationInitializer) Close() error {
	sqlDB, err := ai.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks the database connection
func (ai *ApplicationInitializer) Health(ctx context.Context) *domain.HealthCheckResult {
	if err := ai.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return &domain.HealthCheckResult{
			Status:  domain.HealthStatusUnhealthy,
			Message: fmt.Sprintf("Database health check failed: %v", err),
		}
	}

	return &domain.HealthCheckResult{
		Status:  domain.HealthStatusHealthy,
		Message: "Database is healthy",
	}
}
