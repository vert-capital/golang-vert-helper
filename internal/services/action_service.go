package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// ActionService gerencia o ciclo de vida das actions
type ActionService struct {
	actionRepo        domain.ActionRepository
	actionServiceRepo domain.ActionServiceRepository
	questionRepo      domain.QuestionRepository
	executionRepo     domain.ActionExecutionRepository
	handlers          map[string]domain.ActionHandler
	onExecution       domain.OnActionExecution
	logger            *slog.Logger
}

// NewActionService cria um novo ActionService
func NewActionService(
	actionRepo domain.ActionRepository,
	actionServiceRepo domain.ActionServiceRepository,
	questionRepo domain.QuestionRepository,
	executionRepo domain.ActionExecutionRepository,
	logger *slog.Logger,
) *ActionService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ActionService{
		actionRepo:        actionRepo,
		actionServiceRepo: actionServiceRepo,
		questionRepo:      questionRepo,
		executionRepo:     executionRepo,
		handlers:          make(map[string]domain.ActionHandler),
		logger:            logger,
	}
}

// RegisterHandler registra um handler para um slug de ação
func (s *ActionService) RegisterHandler(slug string, handler domain.ActionHandler) {
	s.handlers[slug] = handler
}

// RegisterActionWithQuestions registra uma action com suas questões e sincroniza no banco
func (s *ActionService) RegisterActionWithQuestions(ctx context.Context, action *domain.Action, questions []domain.Question) error {
	// Busca ou cria a action
	existing, err := s.actionRepo.GetBySlug(ctx, action.Slug)
	if err != nil && err != domain.ErrActionNotFound {
		return err
	}

	if existing == nil {
		// Cria nova action
		action.ID = uuid.New().String()
		if err := s.actionRepo.Create(ctx, action); err != nil {
			return fmt.Errorf("falha ao criar action: %w", err)
		}
		existing = action
	} else {
		// Atualiza action existente
		existing.Title = action.Title
		existing.Description = action.Description
		existing.Active = action.Active
		if err := s.actionRepo.Update(ctx, existing); err != nil {
			return fmt.Errorf("falha ao atualizar action: %w", err)
		}
	}

	// Sincroniza as questões
	if err := s.syncQuestions(ctx, existing.ID, questions); err != nil {
		return fmt.Errorf("falha ao sincronizar questões: %w", err)
	}

	s.logger.Info("action registered", "slug", action.Slug, "action_id", existing.ID)
	return nil
}

// syncQuestions sincroniza as questões de uma action no banco
func (s *ActionService) syncQuestions(ctx context.Context, actionID string, questions []domain.Question) error {
	// Remove todas as questões existentes
	existingQuestions, err := s.questionRepo.ListByActionID(ctx, actionID)
	if err != nil {
		return err
	}

	for _, q := range existingQuestions {
		if err := s.questionRepo.Delete(ctx, q.ID); err != nil {
			s.logger.Error("falha ao deletar questão", "id", q.ID, "error", err)
		}
	}

	// Cria as novas questões
	questionMap := make(map[string]*domain.Question)
	if err := s.createQuestions(ctx, actionID, questions, nil, questionMap); err != nil {
		return err
	}

	return nil
}

// createQuestions cria as questões recursivamente (suporta filhas/children)
func (s *ActionService) createQuestions(ctx context.Context, actionID string, questions []domain.Question, parentID *string, questionMap map[string]*domain.Question) error {
	for i, q := range questions {
		q.ID = uuid.New().String()
		q.ActionID = actionID
		q.ParentID = parentID
		q.Order = i

		if err := s.questionRepo.Create(ctx, &q); err != nil {
			return fmt.Errorf("falha ao criar questão: %w", err)
		}

		questionMap[q.Slug] = &q

		// Se tem filhas, processa recursivamente
		if len(q.Children) > 0 {
			qID := q.ID
			if err := s.createQuestions(ctx, actionID, q.Children, &qID, questionMap); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetOnExecution define o callback chamado após cada execução
func (s *ActionService) SetOnExecution(fn domain.OnActionExecution) {
	s.onExecution = fn
}

// GetAction retorna uma ação pelo slug
func (s *ActionService) GetAction(ctx context.Context, slug string) (*domain.Action, error) {
	return s.actionRepo.GetBySlug(ctx, slug)
}

// ListActions retorna todas as ações de um serviço
// PaginatedActions contém o resultado paginado de actions
type PaginatedActions struct {
	Count      int64            `json:"count"`
	TotalPages int              `json:"total_pages"`
	Results    []*domain.Action `json:"results"`
}

func (s *ActionService) ListActions(ctx context.Context, serviceID string, page, pageSize int) (*PaginatedActions, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	total, err := s.actionRepo.Count(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	var actions []*domain.Action
	if serviceID == "" {
		actions, err = s.actionRepo.ListAll(ctx, pageSize, offset)
	} else {
		actions, err = s.actionRepo.ListByServiceID(ctx, serviceID, pageSize, offset)
	}
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedActions{
		Count:      total,
		TotalPages: totalPages,
		Results:    actions,
	}, nil
}

// Execute executa uma ação com as respostas fornecidas pelo usuário
func (s *ActionService) Execute(ctx context.Context, slug string, input map[string]interface{}) (*domain.ActionResult, error) {
	action, err := s.actionRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !action.Active {
		return nil, fmt.Errorf("%w: %s is inactive", domain.ErrActionNotFound, slug)
	}

	handler, ok := s.handlers[slug]
	if !ok {
		return nil, fmt.Errorf("%w: no handler registered for %s", domain.ErrActionNotFound, slug)
	}

	// Valida as respostas contra as questões da action
	if err := s.validateInput(action, input); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidResponses, err.Error())
	}

	// Persiste execução como "running"
	execution := &domain.ActionExecution{
		ID:       uuid.New().String(),
		ActionID: action.ID,
		Status:   domain.ActionExecutionStatusRunning,
		Input:    marshalJSON(input),
	}
	if err := s.executionRepo.Create(ctx, execution); err != nil {
		s.logger.Error("failed to create execution record", "action", slug, "error", err)
	}

	// Executa o handler
	result, handlerErr := handler(ctx, action, input)

	// Atualiza execução com o resultado
	now := time.Now()
	if handlerErr != nil {
		execution.Status = domain.ActionExecutionStatusFailed
		execution.Error = handlerErr.Error()
		result = &domain.ActionResult{
			Success: false,
			Error:   handlerErr.Error(),
		}
	} else {
		execution.Status = domain.ActionExecutionStatusSuccess
		execution.Output = marshalJSON(result.Data)
	}

	execution.ExecutedAt.Time = now
	execution.ExecutedAt.Valid = true

	if err := s.executionRepo.Update(ctx, execution); err != nil {
		s.logger.Error("failed to update execution record", "action", slug, "error", err)
	}

	// Dispara callback
	if s.onExecution != nil {
		if cbErr := s.onExecution(ctx, execution, result); cbErr != nil {
			s.logger.Error("execution callback returned error", "action", slug, "error", cbErr)
		}
	}

	s.logger.Info("action executed",
		"action", slug,
		"success", result.Success,
	)

	return result, handlerErr
}

// validateInput verifica se todas as questões obrigatórias foram respondidas
func (s *ActionService) validateInput(action *domain.Action, input map[string]interface{}) error {
	for _, q := range action.Questions {
		if !q.Required {
			continue
		}

		// Questões com parent só são obrigatórias se o parent tiver o valor esperado
		if q.ParentID != nil && q.ParentValue != "" {
			parentVal, ok := input[*q.ParentID]
			if !ok || fmt.Sprintf("%v", parentVal) != q.ParentValue {
				continue
			}
		}

		val, ok := input[q.Slug]
		if !ok || val == nil || val == "" {
			return fmt.Errorf("required field missing: %s", q.Slug)
		}
	}
	return nil
}

// marshalJSON converte um valor para JSON string, retornando vazio em caso de erro
func marshalJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
