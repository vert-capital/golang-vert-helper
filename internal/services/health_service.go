package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// HealthService gerencia health checks de serviços
type HealthService struct {
	serviceRepo       domain.ServiceRepository
	serviceHealthRepo domain.ServiceHealthRepository
	checkers          map[string]domain.HealthChecker
	onFailure         domain.OnHealthCheckFailure
	logger            *slog.Logger
}

// NewHealthService cria um novo HealthService
func NewHealthService(
	serviceRepo domain.ServiceRepository,
	serviceHealthRepo domain.ServiceHealthRepository,
	logger *slog.Logger,
) *HealthService {
	if logger == nil {
		logger = slog.Default()
	}
	return &HealthService{
		serviceRepo:       serviceRepo,
		serviceHealthRepo: serviceHealthRepo,
		checkers:          make(map[string]domain.HealthChecker),
		logger:            logger,
	}
}

// RegisterChecker registra um health checker para um serviço
func (s *HealthService) RegisterChecker(serviceName string, checker domain.HealthChecker) {
	s.checkers[serviceName] = checker
}

// RegisterService registra um serviço e sincroniza no banco
func (s *HealthService) RegisterService(ctx context.Context, serviceName, description string) (*domain.Service, error) {
	// Tenta buscar o serviço
	existing, err := s.serviceRepo.GetByName(ctx, serviceName)
	if err != nil && err != domain.ErrServiceNotFound {
		return nil, err
	}

	if existing != nil {
		// Atualiza a descrição se mudou
		if existing.Description != description {
			existing.Description = description
			if err := s.serviceRepo.Update(ctx, existing); err != nil {
				return nil, err
			}
		}
		s.logger.Info("service already registered", "name", serviceName)
		return existing, nil
	}

	// Cria novo serviço
	service := &domain.Service{
		ID:          uuid.New().String(),
		Name:        serviceName,
		Description: description,
		Enabled:     true,
	}

	if err := s.serviceRepo.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("falha ao criar serviço: %w", err)
	}

	s.logger.Info("service registered", "name", serviceName, "service_id", service.ID)
	return service, nil
}

// SetOnFailure define o callback chamado quando um health check falha
func (s *HealthService) SetOnFailure(fn domain.OnHealthCheckFailure) {
	s.onFailure = fn
}

// CheckService executa o health check de um serviço específico e persiste o resultado
func (s *HealthService) CheckService(ctx context.Context, serviceName string) (*domain.HealthCheckResult, error) {
	checker, ok := s.checkers[serviceName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrCheckerNotFound, serviceName)
	}

	result, err := checker.Check(ctx)
	if err != nil {
		result = &domain.HealthCheckResult{
			Status:    domain.HealthStatusUnhealthy,
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	}

	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	// Persiste o resultado se o serviço existe no banco
	service, dbErr := s.serviceRepo.GetByName(ctx, serviceName)
	if dbErr == nil {
		health := &domain.ServiceHealth{
			ID:        uuid.New().String(),
			ServiceID: service.ID,
			Status:    result.Status,
			Message:   result.Message,
			CheckedAt: result.Timestamp,
		}
		if persistErr := s.serviceHealthRepo.Create(ctx, health); persistErr != nil {
			s.logger.Error("failed to persist health check result",
				"service", serviceName,
				"error", persistErr,
			)
		}

		// Dispara callback de falha
		if result.Status == domain.HealthStatusUnhealthy && s.onFailure != nil {
			if cbErr := s.onFailure(ctx, service, result); cbErr != nil {
				s.logger.Error("health check failure callback returned error",
					"service", serviceName,
					"error", cbErr,
				)
			}
		}
	}

	s.logger.Info("health check executed",
		"service", serviceName,
		"status", result.Status,
	)

	return result, nil
}

// CheckAll executa health checks de todos os serviços registrados
func (s *HealthService) CheckAll(ctx context.Context) map[string]*domain.HealthCheckResult {
	results := make(map[string]*domain.HealthCheckResult, len(s.checkers))

	for name := range s.checkers {
		result, err := s.CheckService(ctx, name)
		if err != nil {
			results[name] = &domain.HealthCheckResult{
				Status:    domain.HealthStatusUnknown,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
			continue
		}
		results[name] = result
	}

	return results
}

// GetLatestStatus retorna o último status registrado de um serviço
func (s *HealthService) GetLatestStatus(ctx context.Context, serviceName string) (*domain.ServiceHealth, error) {
	service, err := s.serviceRepo.GetByName(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	return s.serviceHealthRepo.GetLatestByServiceID(ctx, service.ID)
}

// GetAllLatestStatuses retorna o último status de todos os serviços
func (s *HealthService) GetAllLatestStatuses(ctx context.Context) ([]*domain.ServiceHealth, error) {
	return s.serviceHealthRepo.ListAll(ctx)
}

// OverallStatus retorna um status agregado considerando todos os serviços
func (s *HealthService) OverallStatus(ctx context.Context) domain.HealthStatus {
	statuses, err := s.serviceHealthRepo.ListAll(ctx)
	if err != nil || len(statuses) == 0 {
		return domain.HealthStatusUnknown
	}

	for _, h := range statuses {
		if h.Status == domain.HealthStatusUnhealthy {
			return domain.HealthStatusUnhealthy
		}
	}

	for _, h := range statuses {
		if h.Status == domain.HealthStatusDegraded {
			return domain.HealthStatusDegraded
		}
	}

	return domain.HealthStatusHealthy
}
