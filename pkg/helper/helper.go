package helper

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/adapters"
	"github.com/caiofariavert/golang_vert_helper/internal/domain"
	"github.com/caiofariavert/golang_vert_helper/internal/services"
	healthchecks "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

// Helper é o ponto de entrada da biblioteca
type Helper struct {
	db            *gorm.DB
	repos         *adapters.RepositoryFactory
	healthService *services.HealthService
	actionService *services.ActionService
	authService   *services.AuthService
	syncService   *services.SyncService
	workerPool    *healthchecks.WorkerPool
	logger        *slog.Logger
}

// New cria um novo Helper a partir de uma conexão GORM existente
func New(db *gorm.DB, opts ...Option) *Helper {
	logger := slog.Default()

	repos := adapters.NewRepositoryFactory(db)

	h := &Helper{
		db:     db,
		repos:  repos,
		logger: logger,
	}

	h.healthService = services.NewHealthService(
		repos.GetServiceRepository(),
		repos.GetServiceHealthRepository(),
		logger,
	)
	h.actionService = services.NewActionService(
		repos.GetActionRepository(),
		repos.GetActionServiceRepository(),
		repos.GetQuestionRepository(),
		repos.GetActionExecutionRepository(),
		logger,
	)
	h.authService = services.NewAuthService(db, logger)
	h.syncService = services.NewSyncService(
		repos.GetServiceRepository(),
		repos.GetActionRepository(),
		repos.GetQuestionRepository(),
		logger,
	)

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Option é uma função de configuração opcional
type Option func(*Helper)

// WithLogger define o logger
func WithLogger(l *slog.Logger) Option {
	return func(h *Helper) {
		h.logger = l
		h.healthService = services.NewHealthService(
			h.repos.GetServiceRepository(),
			h.repos.GetServiceHealthRepository(),
			l,
		)
		h.actionService = services.NewActionService(
			h.repos.GetActionRepository(),
			h.repos.GetActionServiceRepository(),
			h.repos.GetQuestionRepository(),
			h.repos.GetActionExecutionRepository(),
			l,
		)
		h.authService = services.NewAuthService(h.db, l)
	}
}

// WithOnHealthCheckFailure define callback para falhas de health check
func WithOnHealthCheckFailure(fn domain.OnHealthCheckFailure) Option {
	return func(h *Helper) {
		h.healthService.SetOnFailure(fn)
	}
}

// WithWorkerPool registra um WorkerPool para monitoramento de workers
func WithWorkerPool(pool *healthchecks.WorkerPool) Option {
	return func(h *Helper) {
		h.workerPool = pool
		h.healthService.RegisterChecker("workers", pool)
	}
}

// WithOnActionExecution define callback para execuções de ações
func WithOnActionExecution(fn domain.OnActionExecution) Option {
	return func(h *Helper) {
		h.actionService.SetOnExecution(fn)
	}
}

// RegisterService registra um health checker e sincroniza o serviço no banco
func (h *Helper) RegisterService(name string, checker domain.HealthChecker) error {
	ctx := context.Background()
	_, err := h.healthService.RegisterService(ctx, name, "")
	if err != nil {
		return err
	}
	h.healthService.RegisterChecker(name, checker)
	return nil
}

// RegisterAction registra um handler para uma action e sincroniza no banco
func (h *Helper) RegisterAction(slug string, title, description string, handler domain.ActionHandler, questions []domain.Question) error {
	ctx := context.Background()
	action := &domain.Action{
		Slug:        slug,
		Title:       title,
		Description: description,
		Active:      true,
	}

	if err := h.actionService.RegisterActionWithQuestions(ctx, action, questions); err != nil {
		return err
	}

	h.actionService.RegisterHandler(slug, handler)
	return nil
}

// LinkActionToService vincula uma action a um serviço
// Essa vinculação permite recomendações de ações quando um serviço falha
func (h *Helper) LinkActionToService(ctx context.Context, actionSlug, serviceName string) error {
	action, err := h.actionService.GetAction(ctx, actionSlug)
	if err != nil {
		return err
	}

	service, err := h.repos.GetServiceRepository().GetByName(ctx, serviceName)
	if err != nil {
		return err
	}

	// Verifica se o vínculo já existe
	_, err = h.repos.GetActionServiceRepository().GetByActionAndService(ctx, action.ID, service.ID)
	if err == nil {
		// Vínculo já existe, não é um erro
		return nil
	}

	if err != domain.ErrActionNotFound {
		return err
	}

	// Cria o vínculo
	actionService := &domain.ActionService{
		ID:        uuid.New().String(),
		ActionID:  action.ID,
		ServiceID: service.ID,
	}

	return h.repos.GetActionServiceRepository().Create(ctx, actionService)
}

// Sync sincroniza definições de serviços e actions com o banco
func (h *Helper) Sync(ctx context.Context, defs []services.ServiceDefinition) error {
	return h.syncService.Sync(ctx, defs)
}

// AutoMigrate executa as migrations do schema via GORM para todos os modelos do helper
func (h *Helper) AutoMigrate() error {
	return h.db.AutoMigrate(
		&domain.Service{},
		&domain.ServiceHealth{},
		&domain.Action{},
		&domain.ActionService{},
		&domain.Question{},
		&domain.ActionExecution{},
		&domain.Worker{},
		&domain.WorkerSnapshot{},
		&domain.AuthUser{},
	)
}

// CheckService executa o health check de um serviço pelo nome
func (h *Helper) CheckService(ctx context.Context, name string) (*domain.HealthCheckResult, error) {
	return h.healthService.CheckService(ctx, name)
}

// CheckAll executa health checks de todos os serviços registrados
func (h *Helper) CheckAll(ctx context.Context) map[string]*domain.HealthCheckResult {
	return h.healthService.CheckAll(ctx)
}

// ExecuteAction executa uma action pelo slug com o input fornecido
func (h *Helper) ExecuteAction(ctx context.Context, slug string, input map[string]interface{}) (*domain.ActionResult, error) {
	return h.actionService.Execute(ctx, slug, input)
}

// GetAction retorna uma action pelo slug
func (h *Helper) GetAction(ctx context.Context, slug string) (*domain.Action, error) {
	return h.actionService.GetAction(ctx, slug)
}

// HealthService expõe o serviço de health checks (para o HTTP adapter)
func (h *Helper) HealthService() *services.HealthService {
	return h.healthService
}

// ActionService expõe o serviço de actions (para o HTTP adapter)
func (h *Helper) ActionService() *services.ActionService {
	return h.actionService
}

// DB retorna a conexão GORM
func (h *Helper) DB() *gorm.DB {
	return h.db
}

// Repos retorna o factory de repositories
func (h *Helper) Repos() *adapters.RepositoryFactory {
	return h.repos
}

func (h *Helper) AuthService() *services.AuthService {
	return h.authService
}
