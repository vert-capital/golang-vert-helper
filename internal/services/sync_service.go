package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// SyncService sincroniza as definições de serviços e actions do código com o banco
type SyncService struct {
	serviceRepo  domain.ServiceRepository
	actionRepo   domain.ActionRepository
	questionRepo domain.QuestionRepository
	logger       *slog.Logger
}

// ServiceDefinition define um serviço a ser sincronizado
type ServiceDefinition struct {
	Name        string
	Description string
	Actions     []ActionDefinition
}

// ActionDefinition define uma action a ser sincronizada
type ActionDefinition struct {
	Slug        string
	Title       string
	Description string
	Questions   []QuestionDefinition
}

// QuestionDefinition define uma questão a ser sincronizada
type QuestionDefinition struct {
	Slug        string
	InputType   domain.QuestionInputType
	Label       string
	Placeholder string
	Required    bool
	Options     []string
	ParentSlug  string
	ParentValue string
	Order       int
	Children    []QuestionDefinition
}

// NewSyncService cria um novo SyncService
func NewSyncService(
	serviceRepo domain.ServiceRepository,
	actionRepo domain.ActionRepository,
	questionRepo domain.QuestionRepository,
	logger *slog.Logger,
) *SyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncService{
		serviceRepo:  serviceRepo,
		actionRepo:   actionRepo,
		questionRepo: questionRepo,
		logger:       logger,
	}
}

// Sync sincroniza uma lista de definições de serviços com o banco
func (s *SyncService) Sync(ctx context.Context, definitions []ServiceDefinition) error {
	for _, def := range definitions {
		service, err := s.syncService(ctx, def)
		if err != nil {
			return err
		}

		for _, actionDef := range def.Actions {
			if err := s.syncAction(ctx, service, actionDef); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncService garante que o serviço existe no banco, criando se necessário
func (s *SyncService) syncService(ctx context.Context, def ServiceDefinition) (*domain.Service, error) {
	existing, err := s.serviceRepo.GetByName(ctx, def.Name)
	if err == nil {
		// Atualiza descrição se mudou
		if existing.Description != def.Description {
			existing.Description = def.Description
			if err := s.serviceRepo.Update(ctx, existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	if err != domain.ErrServiceNotFound {
		return nil, err
	}

	// Cria o serviço
	service := &domain.Service{
		ID:          uuid.New().String(),
		Name:        def.Name,
		Description: def.Description,
		Enabled:     true,
	}
	if err := s.serviceRepo.Create(ctx, service); err != nil {
		return nil, err
	}

	s.logger.Info("service synced", "name", def.Name)
	return service, nil
}

// syncAction garante que a action existe no banco, criando ou atualizando se necessário
func (s *SyncService) syncAction(ctx context.Context, service *domain.Service, def ActionDefinition) error {
	existing, err := s.actionRepo.GetBySlug(ctx, def.Slug)
	if err == nil {
		// Atualiza campos se mudaram
		if existing.Title != def.Title || existing.Description != def.Description {
			existing.Title = def.Title
			existing.Description = def.Description
			if err := s.actionRepo.Update(ctx, existing); err != nil {
				return err
			}
		}

		return s.syncQuestions(ctx, existing, def.Questions)
	}

	if err != domain.ErrActionNotFound {
		return err
	}

	// Cria a action
	action := &domain.Action{
		ID:          uuid.New().String(),
		Slug:        def.Slug,
		Title:       def.Title,
		Description: def.Description,
		Active:      true,
	}
	if err := s.actionRepo.Create(ctx, action); err != nil {
		return err
	}

	s.logger.Info("action synced", "slug", def.Slug)
	return s.syncQuestions(ctx, action, def.Questions)
}

// syncQuestions sincroniza as questões de uma action
func (s *SyncService) syncQuestions(ctx context.Context, action *domain.Action, defs []QuestionDefinition) error {
	return s.syncQuestionsWithParent(ctx, action, defs, nil)
}

func (s *SyncService) syncQuestionsWithParent(ctx context.Context, action *domain.Action, defs []QuestionDefinition, parentID *string) error {
	for _, def := range defs {
		existing, err := s.questionRepo.GetByID(ctx, def.Slug)

		if err != nil {
			// Cria questão
			q := &domain.Question{
				ID:          uuid.New().String(),
				ActionID:    action.ID,
				Slug:        def.Slug,
				InputType:   def.InputType,
				Label:       def.Label,
				Placeholder: def.Placeholder,
				Required:    def.Required,
				ParentID:    parentID,
				ParentValue: def.ParentValue,
				Order:       def.Order,
			}
			if err := s.questionRepo.Create(ctx, q); err != nil {
				return err
			}
			existing = q
		} else {
			// Atualiza se mudou
			existing.Label = def.Label
			existing.Required = def.Required
			existing.Order = def.Order
			if err := s.questionRepo.Update(ctx, existing); err != nil {
				return err
			}
		}

		// Sincroniza filhos recursivamente
		if len(def.Children) > 0 {
			if err := s.syncQuestionsWithParent(ctx, action, def.Children, &existing.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
