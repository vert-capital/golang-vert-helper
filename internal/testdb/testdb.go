package testdb

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
)

// TestDB holds a test database connection
type TestDB struct {
	DB *gorm.DB
	t  *testing.T
}

// Setup creates a test database and runs migrations
func Setup(t *testing.T) *TestDB {
	dsn := getTestDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(
		&domain.Service{},
		&domain.ServiceHealth{},
		&domain.Action{},
		&domain.Question{},
		&domain.ActionExecution{},
		&domain.Worker{},
		&domain.WorkerSnapshot{},
	); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return &TestDB{
		DB: db,
		t:  t,
	}
}

// Cleanup cleans up the test database
func (tdb *TestDB) Cleanup() {
	// Delete all tables
	tdb.DB.Migrator().DropTable(
		&domain.WorkerSnapshot{},
		&domain.Worker{},
		&domain.ActionExecution{},
		&domain.Question{},
		&domain.Action{},
		&domain.ServiceHealth{},
		&domain.Service{},
	)

	sqlDB, err := tdb.DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// getTestDSN returns the test database DSN
func getTestDSN() string {
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("TEST_DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("TEST_DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dbname := os.Getenv("TEST_DB_NAME")
	if dbname == "" {
		dbname = "vert_helper_test"
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
}

// CreateService creates a test service
func (tdb *TestDB) CreateService(ctx context.Context, name string) *domain.Service {
	service := &domain.Service{
		Name:    name,
		Enabled: true,
	}

	if err := tdb.DB.WithContext(ctx).Create(service).Error; err != nil {
		tdb.t.Fatalf("failed to create test service: %v", err)
	}

	return service
}

// CreateAction creates a test action
func (tdb *TestDB) CreateAction(ctx context.Context, serviceID, slug, title string) *domain.Action {
	action := &domain.Action{
		Slug:   slug,
		Title:  title,
		Active: true,
	}

	if err := tdb.DB.WithContext(ctx).Create(action).Error; err != nil {
		tdb.t.Fatalf("failed to create test action: %v", err)
	}

	// Se um serviceID foi fornecido, vincula a action ao serviço
	if serviceID != "" {
		actionService := &domain.ActionService{
			ID:        uuid.New().String(),
			ActionID:  action.ID,
			ServiceID: serviceID,
		}
		if err := tdb.DB.WithContext(ctx).Create(actionService).Error; err != nil {
			tdb.t.Fatalf("failed to link action to service: %v", err)
		}
	}

	return action
}

// CreateQuestion creates a test question
func (tdb *TestDB) CreateQuestion(ctx context.Context, actionID, slug string, inputType domain.QuestionInputType) *domain.Question {
	question := &domain.Question{
		ActionID:  actionID,
		Slug:      slug,
		InputType: inputType,
		Label:     slug,
		Required:  false,
	}

	if err := tdb.DB.WithContext(ctx).Create(question).Error; err != nil {
		tdb.t.Fatalf("failed to create test question: %v", err)
	}

	return question
}
