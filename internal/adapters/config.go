package adapters

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// Config holds the configuration for the Vert Helper
type Config struct {
	Database *DatabaseConfig
	Services map[string]domain.HealthChecker
	Actions  map[string]domain.ActionHandler
	Logger   *log.Logger

	// Callbacks
	OnHealthCheckFailure domain.OnHealthCheckFailure
	OnActionExecution    domain.OnActionExecution
	OnWorkerStatusChange domain.OnWorkerStatusChange
}

// DatabaseConfig holds PostgreSQL connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string

	// Connection pool
	MaxOpenConns int
	MaxIdleConns int
}

// NewBuilder creates a new configuration builder
func NewBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{
			Database: &DatabaseConfig{
				Host:         "localhost",
				Port:         5432,
				SSLMode:      "disable",
				MaxOpenConns: 100,
				MaxIdleConns: 10,
			},
			Services: make(map[string]domain.HealthChecker),
			Actions:  make(map[string]domain.ActionHandler),
		},
	}
}

// ConfigBuilder provides a fluent API for building Config
type ConfigBuilder struct {
	config *Config
}

// WithDatabaseHost sets the database host
func (b *ConfigBuilder) WithDatabaseHost(host string) *ConfigBuilder {
	b.config.Database.Host = host
	return b
}

// WithDatabasePort sets the database port
func (b *ConfigBuilder) WithDatabasePort(port int) *ConfigBuilder {
	b.config.Database.Port = port
	return b
}

// WithDatabaseUser sets the database user
func (b *ConfigBuilder) WithDatabaseUser(user string) *ConfigBuilder {
	b.config.Database.User = user
	return b
}

// WithDatabasePassword sets the database password
func (b *ConfigBuilder) WithDatabasePassword(password string) *ConfigBuilder {
	b.config.Database.Password = password
	return b
}

// WithDatabaseName sets the database name
func (b *ConfigBuilder) WithDatabaseName(name string) *ConfigBuilder {
	b.config.Database.DBName = name
	return b
}

// WithDatabaseSSLMode sets the SSL mode
func (b *ConfigBuilder) WithDatabaseSSLMode(sslMode string) *ConfigBuilder {
	b.config.Database.SSLMode = sslMode
	return b
}

// WithDatabase sets the entire database config from a DSN
func (b *ConfigBuilder) WithDatabase(dsn string) *ConfigBuilder {
	b.config.Database.Host = dsn
	return b
}

// WithService registers a health checker for a service
func (b *ConfigBuilder) WithService(name string, checker domain.HealthChecker) *ConfigBuilder {
	b.config.Services[name] = checker
	return b
}

// WithAction registers an action handler
func (b *ConfigBuilder) WithAction(slug string, handler domain.ActionHandler) *ConfigBuilder {
	b.config.Actions[slug] = handler
	return b
}

// WithLogger sets the logger
func (b *ConfigBuilder) WithLogger(log *log.Logger) *ConfigBuilder {
	b.config.Logger = log
	return b
}

// OnHealthCheckFailure sets the callback for health check failures
func (b *ConfigBuilder) OnHealthCheckFailure(cb domain.OnHealthCheckFailure) *ConfigBuilder {
	b.config.OnHealthCheckFailure = cb
	return b
}

// OnActionExecution sets the callback for action executions
func (b *ConfigBuilder) OnActionExecution(cb domain.OnActionExecution) *ConfigBuilder {
	b.config.OnActionExecution = cb
	return b
}

// OnWorkerStatusChange sets the callback for worker status changes
func (b *ConfigBuilder) OnWorkerStatusChange(cb domain.OnWorkerStatusChange) *ConfigBuilder {
	b.config.OnWorkerStatusChange = cb
	return b
}

// Build returns the built Config
func (b *ConfigBuilder) Build() *Config {
	return b.config
}

// GetPostgresDSN returns the PostgreSQL DSN string
func (c *Config) GetPostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// DatabaseAdapter holds the GORM database connection
type DatabaseAdapter struct {
	DB *gorm.DB
}

// NewDatabaseAdapter creates a new PostgreSQL adapter
func NewDatabaseAdapter(dsn string) (*DatabaseAdapter, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)

	return &DatabaseAdapter{DB: db}, nil
}

// Close closes the database connection
func (a *DatabaseAdapter) Close() error {
	sqlDB, err := a.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping verifies the database connection
func (a *DatabaseAdapter) Ping(ctx context.Context) error {
	sqlDB, err := a.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// AutoMigrate runs auto migrations for all models
func (a *DatabaseAdapter) AutoMigrate() error {
	return a.DB.AutoMigrate(
		&domain.Service{},
		&domain.ServiceHealth{},
		&domain.Action{},
		&domain.Question{},
		&domain.ActionExecution{},
		&domain.Worker{},
		&domain.WorkerSnapshot{},
	)
}
