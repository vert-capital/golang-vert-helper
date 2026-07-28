package domain

import "errors"

var (
	// Service errors
	ErrServiceNotFound = errors.New("service not found")
	ErrServiceExists   = errors.New("service already exists")

	// Action errors
	ErrActionNotFound   = errors.New("action not found")
	ErrActionExists     = errors.New("action already exists")
	ErrActionValidation = errors.New("action validation failed")
	ErrInvalidResponses = errors.New("invalid action responses")

	// Question errors
	ErrQuestionNotFound = errors.New("question not found")
	ErrInvalidQuestions = errors.New("invalid questions structure")

	// Health check errors
	ErrHealthCheckFailed = errors.New("health check failed")
	ErrCheckerNotFound   = errors.New("health checker not found")

	// Worker errors
	ErrWorkerNotFound   = errors.New("worker not found")
	ErrWorkerNotRunning = errors.New("worker is not running")

	// Database errors
	ErrDatabaseConnection = errors.New("failed to connect to database")

	// Configuration errors
	ErrInvalidConfig = errors.New("invalid configuration")

	// Auth errors
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserInactive       = errors.New("user inactive")
)
