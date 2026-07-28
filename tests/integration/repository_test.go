package repository_test

import (
	"context"
	"testing"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
	. "github.com/caiofariavert/golang_vert_helper/internal/repository"
	"github.com/caiofariavert/golang_vert_helper/internal/testdb"
)

// TestServiceRepository_Create tests creating a service
func TestServiceRepository_Create(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	repo := NewServiceRepository(db.DB)
	ctx := context.Background()

	service := &domain.Service{
		Name:    "Test Service",
		Enabled: true,
	}

	if err := repo.Create(ctx, service); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if service.ID == "" {
		t.Error("Service ID should not be empty")
	}
}

// TestServiceRepository_GetByID tests retrieving a service by ID
func TestServiceRepository_GetByID(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	repo := NewServiceRepository(db.DB)
	ctx := context.Background()

	// Create a service
	service := db.CreateService(ctx, "Test Service")

	// Get the service
	retrieved, err := repo.GetByID(ctx, service.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != service.ID {
		t.Errorf("Expected ID %s, got %s", service.ID, retrieved.ID)
	}

	if retrieved.Name != service.Name {
		t.Errorf("Expected name %s, got %s", service.Name, retrieved.Name)
	}
}

// TestServiceRepository_GetByName tests retrieving a service by name
func TestServiceRepository_GetByName(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	repo := NewServiceRepository(db.DB)
	ctx := context.Background()

	// Create a service
	service := db.CreateService(ctx, "Unique Service Name")

	// Get by name
	retrieved, err := repo.GetByName(ctx, "Unique Service Name")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}

	if retrieved.ID != service.ID {
		t.Errorf("Expected ID %s, got %s", service.ID, retrieved.ID)
	}
}

// TestServiceRepository_ListAll tests listing all services
func TestServiceRepository_ListAll(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	repo := NewServiceRepository(db.DB)
	ctx := context.Background()

	// Create multiple services
	db.CreateService(ctx, "Service 1")
	db.CreateService(ctx, "Service 2")
	db.CreateService(ctx, "Service 3")

	// List all
	services, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}
}

// TestServiceRepository_Update tests updating a service
func TestServiceRepository_Update(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	repo := NewServiceRepository(db.DB)
	ctx := context.Background()

	// Create a service
	service := db.CreateService(ctx, "Original Name")

	// Update
	service.Name = "Updated Name"
	if err := repo.Update(ctx, service); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	retrieved, _ := repo.GetByID(ctx, service.ID)
	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected name Updated Name, got %s", retrieved.Name)
	}
}

// TestServiceRepository_Delete tests deleting a service
func TestServiceRepository_Delete(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	repo := NewServiceRepository(db.DB)
	ctx := context.Background()

	// Create a service
	service := db.CreateService(ctx, "To Delete")

	// Delete
	if err := repo.Delete(ctx, service.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err := repo.GetByID(ctx, service.ID)
	if err != domain.ErrServiceNotFound {
		t.Errorf("Expected ErrServiceNotFound, got %v", err)
	}
}

// ========== ActionRepository Tests ==========

// TestActionRepository_Create tests creating an action
func TestActionRepository_Create(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	actionRepo := NewActionRepository(db.DB)
	ctx := context.Background()

	action := &domain.Action{
		Slug:   "test-action",
		Title:  "Test Action",
		Active: true,
	}

	if err := actionRepo.Create(ctx, action); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if action.ID == "" {
		t.Error("Action ID should not be empty")
	}
}

// TestActionRepository_GetBySlug tests retrieving an action by slug
func TestActionRepository_GetBySlug(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	actionRepo := NewActionRepository(db.DB)
	ctx := context.Background()

	// Create test data
	service := db.CreateService(ctx, "Test Service")
	action := db.CreateAction(ctx, service.ID, "unique-slug", "Test Action")

	// Get by slug
	retrieved, err := actionRepo.GetBySlug(ctx, "unique-slug")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}

	if retrieved.ID != action.ID {
		t.Errorf("Expected ID %s, got %s", action.ID, retrieved.ID)
	}
}

// TestActionRepository_ListByServiceID tests listing actions by service
func TestActionRepository_ListByServiceID(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	actionRepo := NewActionRepository(db.DB)
	ctx := context.Background()

	// Create test data
	service := db.CreateService(ctx, "Test Service")
	db.CreateAction(ctx, service.ID, "action-1", "Action 1")
	db.CreateAction(ctx, service.ID, "action-2", "Action 2")

	// List
	actions, err := actionRepo.ListByServiceID(ctx, service.ID)
	if err != nil {
		t.Fatalf("ListByServiceID failed: %v", err)
	}

	if len(actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(actions))
	}
}

// ========== QuestionRepository Tests ==========

// TestQuestionRepository_Create tests creating a question
func TestQuestionRepository_Create(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	questionRepo := NewQuestionRepository(db.DB)
	ctx := context.Background()

	// Create test data
	service := db.CreateService(ctx, "Test Service")
	action := db.CreateAction(ctx, service.ID, "test-action", "Test Action")

	question := &domain.Question{
		ActionID:  action.ID,
		Slug:      "test-question",
		InputType: domain.QuestionInputTypeText,
		Label:     "Test Question",
		Required:  true,
	}

	if err := questionRepo.Create(ctx, question); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if question.ID == "" {
		t.Error("Question ID should not be empty")
	}
}

// TestQuestionRepository_ListByActionID tests listing questions by action
func TestQuestionRepository_ListByActionID(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	questionRepo := NewQuestionRepository(db.DB)
	ctx := context.Background()

	// Create test data
	service := db.CreateService(ctx, "Test Service")
	action := db.CreateAction(ctx, service.ID, "test-action", "Test Action")
	db.CreateQuestion(ctx, action.ID, "question-1", domain.QuestionInputTypeText)
	db.CreateQuestion(ctx, action.ID, "question-2", domain.QuestionInputTypeSelect)

	// List
	questions, err := questionRepo.ListByActionID(ctx, action.ID)
	if err != nil {
		t.Fatalf("ListByActionID failed: %v", err)
	}

	if len(questions) != 2 {
		t.Errorf("Expected 2 questions, got %d", len(questions))
	}
}

// ========== WorkerRepository Tests ==========

// TestWorkerRepository_Create tests creating a worker
func TestWorkerRepository_Create(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	workerRepo := NewWorkerRepository(db.DB)
	ctx := context.Background()

	// Create test data
	service := db.CreateService(ctx, "Test Service")

	worker := &domain.Worker{
		ServiceID:      service.ID,
		Name:           "Test Worker",
		Status:         domain.WorkerStatusIdle,
		ProcessedCount: 0,
		FailedCount:    0,
	}

	if err := workerRepo.Create(ctx, worker); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if worker.ID == "" {
		t.Error("Worker ID should not be empty")
	}
}

// TestWorkerRepository_ListByServiceID tests listing workers by service
func TestWorkerRepository_ListByServiceID(t *testing.T) {
	db := testdb.Setup(t)
	defer db.Cleanup()

	workerRepo := NewWorkerRepository(db.DB)
	ctx := context.Background()

	// Create test data
	service := db.CreateService(ctx, "Test Service")

	worker1 := &domain.Worker{
		ServiceID: service.ID,
		Name:      "Worker 1",
		Status:    domain.WorkerStatusRunning,
	}
	worker2 := &domain.Worker{
		ServiceID: service.ID,
		Name:      "Worker 2",
		Status:    domain.WorkerStatusIdle,
	}

	workerRepo.Create(ctx, worker1)
	workerRepo.Create(ctx, worker2)

	// List
	workers, err := workerRepo.ListByServiceID(ctx, service.ID)
	if err != nil {
		t.Fatalf("ListByServiceID failed: %v", err)
	}

	if len(workers) != 2 {
		t.Errorf("Expected 2 workers, got %d", len(workers))
	}
}
