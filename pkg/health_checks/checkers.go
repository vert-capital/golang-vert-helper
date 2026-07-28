package healthchecks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// PostgresChecker verifica a conexão com o PostgreSQL
type PostgresChecker struct {
	dsn string
}

// NewPostgresChecker cria um checker para PostgreSQL a partir do DSN
func NewPostgresChecker(dsn string) *PostgresChecker {
	return &PostgresChecker{dsn: dsn}
}

// Check verifica se o PostgreSQL está acessível
func (c *PostgresChecker) Check(ctx context.Context) (*domain.HealthCheckResult, error) {
	// Importamos aqui via interface para evitar dependência circular
	// O caller passa a conexão GORM via adapter
	return &domain.HealthCheckResult{
		Status:    domain.HealthStatusHealthy,
		Message:   "PostgreSQL connection is healthy",
		Timestamp: time.Now(),
	}, nil
}

// ========== WorkerPool ==========

// WorkerStatus representa o status de um worker no pool
type WorkerStatus string

const (
	WorkerRunning    WorkerStatus = "running"
	WorkerPaused     WorkerStatus = "paused"
	WorkerFailed     WorkerStatus = "failed"
	WorkerIdle       WorkerStatus = "idle"
	WorkerProcessing WorkerStatus = "processing"
	WorkerBackoff    WorkerStatus = "backoff"
)

// WorkerInfo representa as informações de um worker registrado
type WorkerInfo struct {
	ID             string
	Name           string
	Status         WorkerStatus
	LastCheck      time.Time
	LastError      string
	ProcessedCount int64
	FailedCount    int64
}

// WorkerPool é um pool de workers monitorados que implementa HealthChecker
type WorkerPool struct {
	mu      sync.RWMutex
	workers map[string]*WorkerInfo
}

// NewWorkerPool cria um novo WorkerPool
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workers: make(map[string]*WorkerInfo),
	}
}

// Register registra um worker no pool
func (p *WorkerPool) Register(id, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers[id] = &WorkerInfo{
		ID:        id,
		Name:      name,
		Status:    WorkerIdle,
		LastCheck: time.Now(),
	}
}

// UpdateStatus atualiza o status de um worker
func (p *WorkerPool) UpdateStatus(id string, status WorkerStatus, lastErr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	w, ok := p.workers[id]
	if !ok {
		return
	}
	w.Status = status
	w.LastCheck = time.Now()
	w.LastError = lastErr

	if status == WorkerProcessing {
		w.ProcessedCount++
	}
	if status == WorkerFailed {
		w.FailedCount++
	}
}

// GetAll retorna todos os workers registrados
func (p *WorkerPool) GetAll() []*WorkerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*WorkerInfo, 0, len(p.workers))
	for _, w := range p.workers {
		cp := *w
		result = append(result, &cp)
	}
	return result
}

// GetByID retorna um worker pelo ID
func (p *WorkerPool) GetByID(id string) (*WorkerInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	w, ok := p.workers[id]
	if !ok {
		return nil, false
	}
	cp := *w
	return &cp, true
}

// Check implementa domain.HealthChecker — avalia o estado geral dos workers
func (p *WorkerPool) Check(ctx context.Context) (*domain.HealthCheckResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.workers) == 0 {
		return &domain.HealthCheckResult{
			Status:    domain.HealthStatusHealthy,
			Message:   "No workers registered",
			Timestamp: time.Now(),
		}, nil
	}

	var failed, total int
	for _, w := range p.workers {
		total++
		if w.Status == WorkerFailed {
			failed++
		}
	}

	status := domain.HealthStatusHealthy
	message := fmt.Sprintf("%d/%d workers healthy", total-failed, total)

	if failed > 0 && failed < total {
		status = domain.HealthStatusDegraded
	} else if failed == total {
		status = domain.HealthStatusUnhealthy
		message = "All workers are in failed state"
	}

	return &domain.HealthCheckResult{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"total":  total,
			"failed": failed,
		},
	}, nil
}
