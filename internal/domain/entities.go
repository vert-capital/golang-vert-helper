package domain

import (
	"database/sql"
	"time"
)

// ========== Service Entity ==========

// Service representa um serviço monitorado pela aplicação
type Service struct {
	ID          string    `gorm:"primaryKey;type:uuid"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Description string    `gorm:"type:text"`
	Enabled     bool      `gorm:"default:true"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName especifica o nome da tabela
func (Service) TableName() string {
	return "gohelper_services"
}

// ========== ServiceHealth Entity ==========

// ServiceHealth represents the latest health status of a service
type ServiceHealth struct {
	ID        string       `gorm:"primaryKey;type:uuid"`
	ServiceID string       `gorm:"type:uuid;not null;index"`
	Status    HealthStatus `gorm:"type:varchar(50);not null;index"`
	Message   string       `gorm:"type:text"`
	CheckedAt time.Time    `gorm:"autoCreateTime;index"`
	ExpiresAt sql.NullTime `gorm:"index"`
	CreatedAt time.Time    `gorm:"autoCreateTime"`
	UpdatedAt time.Time    `gorm:"autoUpdateTime"`

	// Foreign key
	Service *Service `gorm:"foreignKey:ServiceID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName especifica o nome da tabela
func (ServiceHealth) TableName() string {
	return "gohelper_service_health"
}

// ========== Action Entity ==========

// Action representa uma ação que pode ser executada no contexto de uma questão
type Action struct {
	ID          string     `gorm:"primaryKey;type:uuid"`
	Slug        string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	Title       string     `gorm:"type:varchar(255);not null"`
	Description string     `gorm:"type:text"`
	Active      bool       `gorm:"default:true;index"`
	Questions   []Question `gorm:"foreignKey:ActionID;references:ID"`
	Services    []Service  `gorm:"many2many:gohelper_action_services;"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
}

// TableName especifica o nome da tabela
func (Action) TableName() string {
	return "gohelper_actions"
}

// ========== ActionService Entity (Many-to-Many Junction) ==========

// ActionService representa o vínculo entre uma ação e um serviço
// Uma ação pode ser recomendada quando um ou mais serviços falham
type ActionService struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	ActionID  string    `gorm:"type:uuid;not null;index:idx_action_service,unique"`
	ServiceID string    `gorm:"type:uuid;not null;index:idx_action_service,unique"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Foreign keys with cascade delete
	Action  *Action  `gorm:"foreignKey:ActionID;references:ID;constraint:OnDelete:CASCADE"`
	Service *Service `gorm:"foreignKey:ServiceID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName especifica o nome da tabela
func (ActionService) TableName() string {
	return "gohelper_action_services"
}

// ========== Question Entity ==========

// Question represents a form question that belongs to an action
type Question struct {
	ID          string            `gorm:"primaryKey;type:uuid"`
	ActionID    string            `gorm:"type:uuid;not null;index"`
	Slug        string            `gorm:"type:varchar(255);not null;uniqueIndex:idx_action_question_slug"`
	InputType   QuestionInputType `gorm:"type:varchar(50);not null"`
	Label       string            `gorm:"type:varchar(255);not null"`
	Placeholder string            `gorm:"type:varchar(255)"`
	Required    bool              `gorm:"default:false"`
	Options     string            `gorm:"type:json"` // JSON array of options
	ParentID    *string           `gorm:"type:uuid;index"`
	ParentValue string            `gorm:"type:varchar(255)"`
	Order       int               `gorm:"default:0"`
	CreatedAt   time.Time         `gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `gorm:"autoUpdateTime"`

	// Foreign keys
	Action   *Action    `gorm:"foreignKey:ActionID;references:ID;constraint:OnDelete:CASCADE"`
	Parent   *Question  `gorm:"foreignKey:ParentID;references:ID;constraint:OnDelete:SET NULL"`
	Children []Question `gorm:"foreignKey:ParentID;references:ID"`
}

// TableName especifica o nome da tabela
func (Question) TableName() string {
	return "gohelper_questions"
}

// ========== ActionExecution Entity ==========

// ActionExecution represents the execution of an action
type ActionExecution struct {
	ID         string                `gorm:"primaryKey;type:uuid"`
	ActionID   string                `gorm:"type:uuid;not null;index"`
	Status     ActionExecutionStatus `gorm:"type:varchar(50);not null;index"`
	Input      string                `gorm:"type:json"` // Respostas do usuário
	Output     string                `gorm:"type:json"` // Resultado da execução
	Error      string                `gorm:"type:text"`
	ExecutedAt sql.NullTime
	CreatedAt  time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`

	// Foreign key
	Action *Action `gorm:"foreignKey:ActionID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName especifica o nome da tabela
func (ActionExecution) TableName() string {
	return "gohelper_action_executions"
}

// ========== Worker Entity ==========

// Worker represents a background job/worker being monitored
type Worker struct {
	ID             string       `gorm:"primaryKey;type:uuid"`
	ServiceID      string       `gorm:"type:uuid;not null;index"`
	Name           string       `gorm:"type:varchar(255);not null"`
	Status         WorkerStatus `gorm:"type:varchar(50);not null;index"`
	LastCheck      time.Time    `gorm:"index"`
	LastError      string       `gorm:"type:text"`
	ProcessedCount int64        `gorm:"default:0"`
	FailedCount    int64        `gorm:"default:0"`
	CreatedAt      time.Time    `gorm:"autoCreateTime"`
	UpdatedAt      time.Time    `gorm:"autoUpdateTime"`

	// Foreign key
	Service *Service `gorm:"foreignKey:ServiceID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName especifica o nome da tabela
func (Worker) TableName() string {
	return "gohelper_workers"
}

// ========== WorkerSnapshot Entity ==========

// WorkerSnapshot represents a historical snapshot of worker status
type WorkerSnapshot struct {
	ID             string       `gorm:"primaryKey;type:uuid"`
	ServiceID      string       `gorm:"type:uuid;not null;index"`
	WorkerID       string       `gorm:"type:uuid;not null;index"`
	Status         WorkerStatus `gorm:"type:varchar(50);not null"`
	ErrorMessage   string       `gorm:"type:text"`
	ProcessedCount int64
	FailedCount    int64
	UptimeSeconds  int64
	CreatedAt      time.Time `gorm:"autoCreateTime;index"`
}

// TableName especifica o nome da tabela
func (WorkerSnapshot) TableName() string {
	return "gohelper_worker_snapshots"
}

// ========== AuthUser Entity ==========

// AuthUser representa o usuario de autenticacao das rotas HTTP do helper
type AuthUser struct {
	ID           string    `gorm:"primaryKey;type:uuid"`
	Email        string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null" json:"password,omitempty"`
	Name         string    `gorm:"type:varchar(255);not null"`
	IsSuperuser  bool      `gorm:"default:false;index"`
	Active       bool      `gorm:"default:true;index"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// TableName especifica o nome da tabela
func (AuthUser) TableName() string {
	return "gohelper_auth_users"
}

// ========== Enums ==========

// HealthStatus representa o status de saúde de um serviço
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "OK"
	HealthStatusDegraded  HealthStatus = "FAILED"
	HealthStatusUnhealthy HealthStatus = "FAILED"
	HealthStatusUnknown   HealthStatus = "UNKNOWN"
)

// QuestionInputType representa o tipo de input de uma questão
type QuestionInputType string

const (
	QuestionInputTypeText     QuestionInputType = "text"
	QuestionInputTypeSelect   QuestionInputType = "select"
	QuestionInputTypeRadio    QuestionInputType = "radio"
	QuestionInputTypeCheckbox QuestionInputType = "checkbox"
	QuestionInputTypeTextarea QuestionInputType = "textarea"
	QuestionInputTypeNumber   QuestionInputType = "number"
	QuestionInputTypeEmail    QuestionInputType = "email"
)

// ActionExecutionStatus representa o status de execução de uma ação
type ActionExecutionStatus string

const (
	ActionExecutionStatusPending   ActionExecutionStatus = "pending"
	ActionExecutionStatusRunning   ActionExecutionStatus = "running"
	ActionExecutionStatusSuccess   ActionExecutionStatus = "success"
	ActionExecutionStatusFailed    ActionExecutionStatus = "failed"
	ActionExecutionStatusCancelled ActionExecutionStatus = "cancelled"
)

// WorkerStatus representa o status de um worker
type WorkerStatus string

const (
	WorkerStatusRunning    WorkerStatus = "running"
	WorkerStatusPaused     WorkerStatus = "paused"
	WorkerStatusFailed     WorkerStatus = "failed"
	WorkerStatusIdle       WorkerStatus = "idle"
	WorkerStatusProcessing WorkerStatus = "processing"
	WorkerStatusBackoff    WorkerStatus = "backoff"
)

// ========== Value Objects ==========

// HealthCheckResult representa o resultado de um health check
type HealthCheckResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ActionResult representa o resultado da execução de uma ação
type ActionResult struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// WorkerDetail representa detalhes de um worker para exposição via API
type WorkerDetail struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Status         WorkerStatus `json:"status"`
	LastCheck      time.Time    `json:"last_check"`
	LastError      string       `json:"last_error,omitempty"`
	ProcessedCount int64        `json:"processed_count"`
	FailedCount    int64        `json:"failed_count"`
	HealthPercent  float64      `json:"health_percent"`
}
