package healthchecks

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// GormPostgresChecker verifica a conexão com o banco via instância GORM existente
type GormPostgresChecker struct {
	db *gorm.DB
}

// NewGormPostgresChecker cria um checker que usa a conexão GORM já estabelecida
func NewGormPostgresChecker(db *gorm.DB) *GormPostgresChecker {
	return &GormPostgresChecker{db: db}
}

// Check executa um SELECT 1 para verificar a conectividade com o banco
func (c *GormPostgresChecker) Check(ctx context.Context) (*domain.HealthCheckResult, error) {
	sqlDB, err := c.db.DB()
	if err != nil {
		return &domain.HealthCheckResult{
			Status:    domain.HealthStatusUnhealthy,
			Message:   fmt.Sprintf("failed to get sql.DB: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return &domain.HealthCheckResult{
			Status:    domain.HealthStatusUnhealthy,
			Message:   fmt.Sprintf("database ping failed: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	stats := sqlDB.Stats()
	return &domain.HealthCheckResult{
		Status:    domain.HealthStatusHealthy,
		Message:   "PostgreSQL connection is healthy",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
		},
	}, nil
}
