package domain

import (
	"context"
)

// ========== Repository Contracts ==========

// ServiceRepository define as operações de persistência para Service
type ServiceRepository interface {
	// Create insere um novo serviço
	Create(ctx context.Context, service *Service) error

	// GetByID recupera um serviço pelo ID
	GetByID(ctx context.Context, id string) (*Service, error)

	// GetByName recupera um serviço pelo nome
	GetByName(ctx context.Context, name string) (*Service, error)

	// ListAll retorna todos os serviços
	ListAll(ctx context.Context) ([]*Service, error)

	// Update atualiza um serviço
	Update(ctx context.Context, service *Service) error

	// Delete remove um serviço
	Delete(ctx context.Context, id string) error
}

// ServiceHealthRepository define as operações de persistência para ServiceHealth
type ServiceHealthRepository interface {
	// Create insere um novo registro de saúde
	Create(ctx context.Context, health *ServiceHealth) error

	// GetLatestByServiceID retorna o status mais recente de um serviço
	GetLatestByServiceID(ctx context.Context, serviceID string) (*ServiceHealth, error)

	// ListByServiceID retorna histórico de saúde de um serviço
	ListByServiceID(ctx context.Context, serviceID string, limit int) ([]*ServiceHealth, error)

	// ListAll retorna o status mais recente de todos os serviços
	ListAll(ctx context.Context) ([]*ServiceHealth, error)
}

// ActionRepository define as operações de persistência para Action
type ActionRepository interface {
	// Create insere uma nova ação
	Create(ctx context.Context, action *Action) error

	// GetByID recupera uma ação pelo ID
	GetByID(ctx context.Context, id string) (*Action, error)

	// GetBySlug recupera uma ação pelo slug
	GetBySlug(ctx context.Context, slug string) (*Action, error)

	// ListByServiceID retorna ações vinculadas a um serviço com paginação
	ListByServiceID(ctx context.Context, serviceID string, limit, offset int) ([]*Action, error)

	// ListAll retorna ações com paginação
	ListAll(ctx context.Context, limit, offset int) ([]*Action, error)

	// Count retorna o total de ações; serviceID vazio conta todas
	Count(ctx context.Context, serviceID string) (int64, error)

	// Update atualiza uma ação
	Update(ctx context.Context, action *Action) error

	// Delete remove uma ação
	Delete(ctx context.Context, id string) error
}

// ActionServiceRepository define as operações de persistência para ActionService (junção)
type ActionServiceRepository interface {
	// Create insere um vínculo entre action e service
	Create(ctx context.Context, actionService *ActionService) error

	// GetByActionAndService recupera um vínculo específico
	GetByActionAndService(ctx context.Context, actionID, serviceID string) (*ActionService, error)

	// ListByActionID retorna todos os serviços vinculados a uma ação
	ListByActionID(ctx context.Context, actionID string) ([]*ActionService, error)

	// ListByServiceID retorna todas as ações vinculadas a um serviço
	ListByServiceID(ctx context.Context, serviceID string) ([]*ActionService, error)

	// Delete remove um vínculo
	Delete(ctx context.Context, actionID, serviceID string) error
}

// QuestionRepository define as operações de persistência para Question
type QuestionRepository interface {
	// Create insere uma nova questão
	Create(ctx context.Context, question *Question) error

	// GetByID recupera uma questão pelo ID
	GetByID(ctx context.Context, id string) (*Question, error)

	// ListByActionID retorna todas as questões de uma ação
	ListByActionID(ctx context.Context, actionID string) ([]*Question, error)

	// Update atualiza uma questão
	Update(ctx context.Context, question *Question) error

	// Delete remove uma questão
	Delete(ctx context.Context, id string) error
}

// ActionExecutionRepository define as operações de persistência para ActionExecution
type ActionExecutionRepository interface {
	// Create insere uma execução
	Create(ctx context.Context, execution *ActionExecution) error

	// GetByID recupera uma execução pelo ID
	GetByID(ctx context.Context, id string) (*ActionExecution, error)

	// ListByActionID retorna o histórico de execuções de uma ação
	ListByActionID(ctx context.Context, actionID string, limit int) ([]*ActionExecution, error)

	// Update atualiza uma execução
	Update(ctx context.Context, execution *ActionExecution) error
}

// WorkerRepository define as operações de persistência para Worker
type WorkerRepository interface {
	// Create insere um novo worker
	Create(ctx context.Context, worker *Worker) error

	// GetByID recupera um worker pelo ID
	GetByID(ctx context.Context, id string) (*Worker, error)

	// ListByServiceID retorna todos os workers de um serviço
	ListByServiceID(ctx context.Context, serviceID string) ([]*Worker, error)

	// Update atualiza um worker
	Update(ctx context.Context, worker *Worker) error

	// Delete remove um worker
	Delete(ctx context.Context, id string) error
}

// WorkerSnapshotRepository define as operações de persistência para WorkerSnapshot
type WorkerSnapshotRepository interface {
	// Create insere um snapshot
	Create(ctx context.Context, snapshot *WorkerSnapshot) error

	// ListByWorkerID retorna histórico de snapshots de um worker
	ListByWorkerID(ctx context.Context, workerID string, limit int) ([]*WorkerSnapshot, error)

	// ListByServiceID retorna snapshots de todos os workers de um serviço
	ListByServiceID(ctx context.Context, serviceID string, limit int) ([]*WorkerSnapshot, error)
}

// ========== Service Contracts ==========

// HealthChecker define a interface para verificação de saúde de um serviço
type HealthChecker interface {
	// Check executa uma verificação de saúde
	// Retorna um HealthCheckResult com o status atual
	Check(ctx context.Context) (*HealthCheckResult, error)
}

// ActionHandler define a assinatura das funções que lidam com ações
type ActionHandler func(ctx context.Context, action *Action, input map[string]interface{}) (*ActionResult, error)

// SyncStrategy define a estratégia de sincronização de dados
type SyncStrategy interface {
	// Sync executa a sincronização
	Sync(ctx context.Context) error
}

// EventPublisher define a interface para publicação de eventos
type EventPublisher interface {
	// Publish publica um evento
	Publish(ctx context.Context, topic string, data interface{}) error
}

// OnHealthCheckFailure é uma função callback para quando um health check falha
type OnHealthCheckFailure func(ctx context.Context, service *Service, result *HealthCheckResult) error

// OnActionExecution é uma função callback para quando uma ação é executada
type OnActionExecution func(ctx context.Context, execution *ActionExecution, result *ActionResult) error

// OnWorkerStatusChange é uma função callback para quando o status de um worker muda
type OnWorkerStatusChange func(ctx context.Context, worker *Worker, oldStatus, newStatus WorkerStatus) error
