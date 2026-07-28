package adapters

import (
	"gorm.io/gorm"

	"github.com/caiofariavert/golang_vert_helper/internal/domain"
	"github.com/caiofariavert/golang_vert_helper/internal/repository"
)

// RepositoryFactory creates all repositories from a GORM connection
type RepositoryFactory struct {
	serviceRepo         domain.ServiceRepository
	serviceHealthRepo   domain.ServiceHealthRepository
	actionRepo          domain.ActionRepository
	questionRepo        domain.QuestionRepository
	actionExecutionRepo domain.ActionExecutionRepository
	actionServiceRepo   domain.ActionServiceRepository
	workerRepo          domain.WorkerRepository
	workerSnapshotRepo  domain.WorkerSnapshotRepository
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(db *gorm.DB) *RepositoryFactory {
	return &RepositoryFactory{
		serviceRepo:         repository.NewServiceRepository(db),
		serviceHealthRepo:   repository.NewServiceHealthRepository(db),
		actionRepo:          repository.NewActionRepository(db),
		questionRepo:        repository.NewQuestionRepository(db),
		actionExecutionRepo: repository.NewActionExecutionRepository(db),
		actionServiceRepo:   repository.NewActionServiceRepository(db),
		workerRepo:          repository.NewWorkerRepository(db),
		workerSnapshotRepo:  repository.NewWorkerSnapshotRepository(db),
	}
}

// GetServiceRepository returns the service repository
func (rf *RepositoryFactory) GetServiceRepository() domain.ServiceRepository {
	return rf.serviceRepo
}

// GetServiceHealthRepository returns the service health repository
func (rf *RepositoryFactory) GetServiceHealthRepository() domain.ServiceHealthRepository {
	return rf.serviceHealthRepo
}

// GetActionRepository returns the action repository
func (rf *RepositoryFactory) GetActionRepository() domain.ActionRepository {
	return rf.actionRepo
}

// GetQuestionRepository returns the question repository
func (rf *RepositoryFactory) GetQuestionRepository() domain.QuestionRepository {
	return rf.questionRepo
}

// GetActionExecutionRepository returns the action execution repository
func (rf *RepositoryFactory) GetActionExecutionRepository() domain.ActionExecutionRepository {
	return rf.actionExecutionRepo
}

// GetActionServiceRepository returns the action service repository
func (rf *RepositoryFactory) GetActionServiceRepository() domain.ActionServiceRepository {
	return rf.actionServiceRepo
}

// GetWorkerRepository returns the worker repository
func (rf *RepositoryFactory) GetWorkerRepository() domain.WorkerRepository {
	return rf.workerRepo
}

// GetWorkerSnapshotRepository returns the worker snapshot repository
func (rf *RepositoryFactory) GetWorkerSnapshotRepository() domain.WorkerSnapshotRepository {
	return rf.workerSnapshotRepo
}
