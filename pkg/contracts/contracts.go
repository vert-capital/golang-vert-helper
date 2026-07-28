package contracts

import (
	internaldomain "github.com/caiofariavert/golang_vert_helper/internal/domain"
	internalservices "github.com/caiofariavert/golang_vert_helper/internal/services"
)

// Tipos de dominio expostos publicamente
// via aliases para manter compatibilidade com a API atual.
type Service = internaldomain.Service
type ServiceHealth = internaldomain.ServiceHealth
type Action = internaldomain.Action
type Question = internaldomain.Question
type ActionExecution = internaldomain.ActionExecution
type Worker = internaldomain.Worker
type WorkerSnapshot = internaldomain.WorkerSnapshot

type HealthStatus = internaldomain.HealthStatus
type QuestionInputType = internaldomain.QuestionInputType
type ActionExecutionStatus = internaldomain.ActionExecutionStatus
type WorkerStatus = internaldomain.WorkerStatus

type HealthCheckResult = internaldomain.HealthCheckResult
type ActionResult = internaldomain.ActionResult
type WorkerDetail = internaldomain.WorkerDetail

type HealthChecker = internaldomain.HealthChecker
type ActionHandler = internaldomain.ActionHandler
type OnHealthCheckFailure = internaldomain.OnHealthCheckFailure
type OnActionExecution = internaldomain.OnActionExecution
type OnWorkerStatusChange = internaldomain.OnWorkerStatusChange

// Definicoes de sincronizacao expostas publicamente.
type ServiceDefinition = internalservices.ServiceDefinition
type ActionDefinition = internalservices.ActionDefinition
type QuestionDefinition = internalservices.QuestionDefinition

const (
	HealthStatusHealthy   = internaldomain.HealthStatusHealthy
	HealthStatusDegraded  = internaldomain.HealthStatusDegraded
	HealthStatusUnhealthy = internaldomain.HealthStatusUnhealthy
	HealthStatusUnknown   = internaldomain.HealthStatusUnknown
)

const (
	QuestionInputTypeText     = internaldomain.QuestionInputTypeText
	QuestionInputTypeSelect   = internaldomain.QuestionInputTypeSelect
	QuestionInputTypeRadio    = internaldomain.QuestionInputTypeRadio
	QuestionInputTypeCheckbox = internaldomain.QuestionInputTypeCheckbox
	QuestionInputTypeTextarea = internaldomain.QuestionInputTypeTextarea
	QuestionInputTypeNumber   = internaldomain.QuestionInputTypeNumber
	QuestionInputTypeEmail    = internaldomain.QuestionInputTypeEmail
)

const (
	ActionExecutionStatusPending   = internaldomain.ActionExecutionStatusPending
	ActionExecutionStatusRunning   = internaldomain.ActionExecutionStatusRunning
	ActionExecutionStatusSuccess   = internaldomain.ActionExecutionStatusSuccess
	ActionExecutionStatusFailed    = internaldomain.ActionExecutionStatusFailed
	ActionExecutionStatusCancelled = internaldomain.ActionExecutionStatusCancelled
)

const (
	WorkerStatusRunning    = internaldomain.WorkerStatusRunning
	WorkerStatusPaused     = internaldomain.WorkerStatusPaused
	WorkerStatusFailed     = internaldomain.WorkerStatusFailed
	WorkerStatusIdle       = internaldomain.WorkerStatusIdle
	WorkerStatusProcessing = internaldomain.WorkerStatusProcessing
	WorkerStatusBackoff    = internaldomain.WorkerStatusBackoff
)
