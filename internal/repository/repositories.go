package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// ServiceRepository implements domain.ServiceRepository using GORM
type ServiceRepository struct {
	db *gorm.DB
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *gorm.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create inserts a new service
func (r *ServiceRepository) Create(ctx context.Context, service *domain.Service) error {
	if service.ID == "" {
		service.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(service)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrServiceExists
		}
		return result.Error
	}
	return nil
}

// GetByID retrieves a service by ID
func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*domain.Service, error) {
	var service domain.Service
	result := r.db.WithContext(ctx).First(&service, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrServiceNotFound
		}
		return nil, result.Error
	}
	return &service, nil
}

// GetByName retrieves a service by name
func (r *ServiceRepository) GetByName(ctx context.Context, name string) (*domain.Service, error) {
	var service domain.Service
	result := r.db.WithContext(ctx).First(&service, "name = ?", name)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrServiceNotFound
		}
		return nil, result.Error
	}
	return &service, nil
}

// ListAll returns all services
func (r *ServiceRepository) ListAll(ctx context.Context) ([]*domain.Service, error) {
	var services []*domain.Service
	result := r.db.WithContext(ctx).Find(&services)
	if result.Error != nil {
		return nil, result.Error
	}
	return services, nil
}

// Update updates a service
func (r *ServiceRepository) Update(ctx context.Context, service *domain.Service) error {
	result := r.db.WithContext(ctx).Save(service)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrServiceNotFound
	}
	return nil
}

// Delete removes a service
func (r *ServiceRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Service{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrServiceNotFound
	}
	return nil
}

// ========== ServiceHealthRepository ==========

// ServiceHealthRepository implements domain.ServiceHealthRepository using GORM
type ServiceHealthRepository struct {
	db *gorm.DB
}

// NewServiceHealthRepository creates a new service health repository
func NewServiceHealthRepository(db *gorm.DB) *ServiceHealthRepository {
	return &ServiceHealthRepository{db: db}
}

// Create inserts a new health record
func (r *ServiceHealthRepository) Create(ctx context.Context, health *domain.ServiceHealth) error {
	if health.ID == "" {
		health.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(health)
	return result.Error
}

// GetLatestByServiceID returns the latest health status
func (r *ServiceHealthRepository) GetLatestByServiceID(ctx context.Context, serviceID string) (*domain.ServiceHealth, error) {
	var health domain.ServiceHealth
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Order("checked_at DESC").
		First(&health)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrServiceNotFound
		}
		return nil, result.Error
	}
	return &health, nil
}

// ListByServiceID returns health history
func (r *ServiceHealthRepository) ListByServiceID(ctx context.Context, serviceID string, limit int) ([]*domain.ServiceHealth, error) {
	var healths []*domain.ServiceHealth
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Order("checked_at DESC").
		Limit(limit).
		Find(&healths)
	if result.Error != nil {
		return nil, result.Error
	}
	return healths, nil
}

// ListAll returns the latest status of all services
func (r *ServiceHealthRepository) ListAll(ctx context.Context) ([]*domain.ServiceHealth, error) {
	var healths []*domain.ServiceHealth
	result := r.db.WithContext(ctx).
		Preload("Service").
		Order("checked_at DESC").
		Find(&healths)
	if result.Error != nil {
		return nil, result.Error
	}
	return healths, nil
}

// ========== ActionRepository ==========

// ActionRepository implements domain.ActionRepository using GORM
type ActionRepository struct {
	db *gorm.DB
}

// NewActionRepository creates a new action repository
func NewActionRepository(db *gorm.DB) *ActionRepository {
	return &ActionRepository{db: db}
}

// Create inserts a new action
func (r *ActionRepository) Create(ctx context.Context, action *domain.Action) error {
	if action.ID == "" {
		action.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(action)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrActionExists
		}
		return result.Error
	}
	return nil
}

// GetByID retrieves an action by ID
func (r *ActionRepository) GetByID(ctx context.Context, id string) (*domain.Action, error) {
	var action domain.Action
	result := r.db.WithContext(ctx).
		Preload("Questions").
		First(&action, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrActionNotFound
		}
		return nil, result.Error
	}
	return &action, nil
}

// GetBySlug retrieves an action by slug
func (r *ActionRepository) GetBySlug(ctx context.Context, slug string) (*domain.Action, error) {
	var action domain.Action
	result := r.db.WithContext(ctx).
		Preload("Questions").
		First(&action, "slug = ?", slug)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrActionNotFound
		}
		return nil, result.Error
	}
	return &action, nil
}

// ListByServiceID returns actions linked to a service with pagination
func (r *ActionRepository) ListByServiceID(ctx context.Context, serviceID string, limit, offset int) ([]*domain.Action, error) {
	var actions []*domain.Action
	result := r.db.WithContext(ctx).
		Preload("Questions").
		Joins("JOIN gohelper_action_services ON gohelper_action_services.action_id = gohelper_actions.id").
		Where("gohelper_action_services.service_id = ?", serviceID).
		Limit(limit).
		Offset(offset).
		Find(&actions)
	if result.Error != nil {
		return nil, result.Error
	}
	return actions, nil
}

// ListAll returns all actions with pagination
func (r *ActionRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.Action, error) {
	var actions []*domain.Action
	result := r.db.WithContext(ctx).
		Preload("Questions").
		Limit(limit).
		Offset(offset).
		Find(&actions)
	if result.Error != nil {
		return nil, result.Error
	}
	return actions, nil
}

// Count returns total number of actions; empty serviceID counts all
func (r *ActionRepository) Count(ctx context.Context, serviceID string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Action{})
	if serviceID != "" {
		query = query.
			Joins("JOIN gohelper_action_services ON gohelper_action_services.action_id = gohelper_actions.id").
			Where("gohelper_action_services.service_id = ?", serviceID)
	}
	result := query.Count(&count)
	return count, result.Error
}

// Update updates an action
func (r *ActionRepository) Update(ctx context.Context, action *domain.Action) error {
	result := r.db.WithContext(ctx).Save(action)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrActionNotFound
	}
	return nil
}

// Delete removes an action
func (r *ActionRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Action{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrActionNotFound
	}
	return nil
}

// ========== QuestionRepository ==========

// QuestionRepository implements domain.QuestionRepository using GORM
type QuestionRepository struct {
	db *gorm.DB
}

// NewQuestionRepository creates a new question repository
func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// Create inserts a new question
func (r *QuestionRepository) Create(ctx context.Context, question *domain.Question) error {
	if question.ID == "" {
		question.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(question)
	return result.Error
}

// GetByID retrieves a question by ID
func (r *QuestionRepository) GetByID(ctx context.Context, id string) (*domain.Question, error) {
	var question domain.Question
	result := r.db.WithContext(ctx).
		Preload("Children").
		First(&question, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrQuestionNotFound
		}
		return nil, result.Error
	}
	return &question, nil
}

// ListByActionID returns all questions for an action
func (r *QuestionRepository) ListByActionID(ctx context.Context, actionID string) ([]*domain.Question, error) {
	var questions []*domain.Question
	result := r.db.WithContext(ctx).
		Where("action_id = ?", actionID).
		Where("parent_id IS NULL").
		Order("`order` ASC").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("`order` ASC")
		}).
		Find(&questions)
	if result.Error != nil {
		return nil, result.Error
	}
	return questions, nil
}

// Update updates a question
func (r *QuestionRepository) Update(ctx context.Context, question *domain.Question) error {
	result := r.db.WithContext(ctx).Save(question)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrQuestionNotFound
	}
	return nil
}

// Delete removes a question
func (r *QuestionRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Question{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrQuestionNotFound
	}
	return nil
}

// ========== ActionExecutionRepository ==========

// ActionExecutionRepository implements domain.ActionExecutionRepository using GORM
type ActionExecutionRepository struct {
	db *gorm.DB
}

// NewActionExecutionRepository creates a new action execution repository
func NewActionExecutionRepository(db *gorm.DB) *ActionExecutionRepository {
	return &ActionExecutionRepository{db: db}
}

// Create inserts a new execution
func (r *ActionExecutionRepository) Create(ctx context.Context, execution *domain.ActionExecution) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(execution)
	return result.Error
}

// GetByID retrieves an execution by ID
func (r *ActionExecutionRepository) GetByID(ctx context.Context, id string) (*domain.ActionExecution, error) {
	var execution domain.ActionExecution
	result := r.db.WithContext(ctx).First(&execution, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrActionNotFound
		}
		return nil, result.Error
	}
	return &execution, nil
}

// ListByActionID returns execution history
func (r *ActionExecutionRepository) ListByActionID(ctx context.Context, actionID string, limit int) ([]*domain.ActionExecution, error) {
	var executions []*domain.ActionExecution
	result := r.db.WithContext(ctx).
		Where("action_id = ?", actionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&executions)
	if result.Error != nil {
		return nil, result.Error
	}
	return executions, nil
}

// Update updates an execution
func (r *ActionExecutionRepository) Update(ctx context.Context, execution *domain.ActionExecution) error {
	result := r.db.WithContext(ctx).Save(execution)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrActionNotFound
	}
	return nil
}

// ========== WorkerRepository ==========

// WorkerRepository implements domain.WorkerRepository using GORM
type WorkerRepository struct {
	db *gorm.DB
}

// NewWorkerRepository creates a new worker repository
func NewWorkerRepository(db *gorm.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

// Create inserts a new worker
func (r *WorkerRepository) Create(ctx context.Context, worker *domain.Worker) error {
	if worker.ID == "" {
		worker.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(worker)
	return result.Error
}

// GetByID retrieves a worker by ID
func (r *WorkerRepository) GetByID(ctx context.Context, id string) (*domain.Worker, error) {
	var worker domain.Worker
	result := r.db.WithContext(ctx).First(&worker, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrWorkerNotFound
		}
		return nil, result.Error
	}
	return &worker, nil
}

// ListByServiceID returns all workers for a service
func (r *WorkerRepository) ListByServiceID(ctx context.Context, serviceID string) ([]*domain.Worker, error) {
	var workers []*domain.Worker
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Find(&workers)
	if result.Error != nil {
		return nil, result.Error
	}
	return workers, nil
}

// Update updates a worker
func (r *WorkerRepository) Update(ctx context.Context, worker *domain.Worker) error {
	result := r.db.WithContext(ctx).Save(worker)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrWorkerNotFound
	}
	return nil
}

// Delete removes a worker
func (r *WorkerRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Worker{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrWorkerNotFound
	}
	return nil
}

// ========== WorkerSnapshotRepository ==========

// WorkerSnapshotRepository implements domain.WorkerSnapshotRepository using GORM
type WorkerSnapshotRepository struct {
	db *gorm.DB
}

// NewWorkerSnapshotRepository creates a new worker snapshot repository
func NewWorkerSnapshotRepository(db *gorm.DB) *WorkerSnapshotRepository {
	return &WorkerSnapshotRepository{db: db}
}

// Create inserts a new snapshot
func (r *WorkerSnapshotRepository) Create(ctx context.Context, snapshot *domain.WorkerSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(snapshot)
	return result.Error
}

// ListByWorkerID returns snapshot history for a worker
func (r *WorkerSnapshotRepository) ListByWorkerID(ctx context.Context, workerID string, limit int) ([]*domain.WorkerSnapshot, error) {
	var snapshots []*domain.WorkerSnapshot
	result := r.db.WithContext(ctx).
		Where("worker_id = ?", workerID).
		Order("created_at DESC").
		Limit(limit).
		Find(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}
	return snapshots, nil
}

// ListByServiceID returns snapshots for all workers in a service
func (r *WorkerSnapshotRepository) ListByServiceID(ctx context.Context, serviceID string, limit int) ([]*domain.WorkerSnapshot, error) {
	var snapshots []*domain.WorkerSnapshot
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Order("created_at DESC").
		Limit(limit).
		Find(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}
	return snapshots, nil
}

// ========== ActionServiceRepository ==========

// ActionServiceRepository implements domain.ActionServiceRepository using GORM
type ActionServiceRepository struct {
	db *gorm.DB
}

// NewActionServiceRepository creates a new action service repository
func NewActionServiceRepository(db *gorm.DB) *ActionServiceRepository {
	return &ActionServiceRepository{db: db}
}

// Create inserts a new link between action and service
func (r *ActionServiceRepository) Create(ctx context.Context, actionService *domain.ActionService) error {
	if actionService.ID == "" {
		actionService.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(actionService)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			// Link já existe, não é um erro
			return nil
		}
		return result.Error
	}
	return nil
}

// GetByActionAndService retrieves a specific link
func (r *ActionServiceRepository) GetByActionAndService(ctx context.Context, actionID, serviceID string) (*domain.ActionService, error) {
	var actionService domain.ActionService
	result := r.db.WithContext(ctx).
		Where("action_id = ? AND service_id = ?", actionID, serviceID).
		First(&actionService)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrActionNotFound
		}
		return nil, result.Error
	}
	return &actionService, nil
}

// ListByActionID returns all services linked to an action
func (r *ActionServiceRepository) ListByActionID(ctx context.Context, actionID string) ([]*domain.ActionService, error) {
	var links []*domain.ActionService
	result := r.db.WithContext(ctx).
		Where("action_id = ?", actionID).
		Find(&links)
	if result.Error != nil {
		return nil, result.Error
	}
	return links, nil
}

// ListByServiceID returns all actions linked to a service
func (r *ActionServiceRepository) ListByServiceID(ctx context.Context, serviceID string) ([]*domain.ActionService, error) {
	var links []*domain.ActionService
	result := r.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Find(&links)
	if result.Error != nil {
		return nil, result.Error
	}
	return links, nil
}

// Delete removes a link
func (r *ActionServiceRepository) Delete(ctx context.Context, actionID, serviceID string) error {
	result := r.db.WithContext(ctx).
		Where("action_id = ? AND service_id = ?", actionID, serviceID).
		Delete(&domain.ActionService{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
