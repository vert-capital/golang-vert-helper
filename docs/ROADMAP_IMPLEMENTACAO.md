# Roadmap de Implementação - Vert Helper Go

---

## 📅 Cronograma (4-5 semanas)

### **Semana 1: Setup e Domain**

#### Objetivos
- ✅ Estrutura Go pronta
- ✅ Models/entities definidos
- ✅ Interfaces/contracts definidos
- ✅ Migrations SQL criadas
- ✅ Projeto compilando

#### Tarefas

**1.1 Setup Inicial**
```bash
# Criar estrutura de pastas
mkdir -p cmd/examples/{basic,with_scheduler,with_custom_checks}
mkdir -p pkg/helper pkg/health_checks pkg/errors
mkdir -p internal/{domain,adapters,services,repository}
mkdir -p migrations tests/{unit,integration}
mkdir -p docs scripts

# Inicializar módulo Go
go mod init github.com/caiofariavert/golang_vert_helper
go mod tidy
```

**Deliverable:** Projeto compila com `go build ./...`

---

**1.2 Domain Entities** 
Arquivo: `internal/domain/entities.go`

```go
// Service
type Service struct {
  ID        uuid.UUID
  Name      string
  IsActive  bool
  CreatedAt time.Time
  UpdatedAt time.Time
}

// ServiceHealth
type ServiceHealth struct {
  ID        uuid.UUID
  ServiceID uuid.UUID
  Status    string // OK, FAILED, UNKNOWN
  Message   *string
  CheckedAt time.Time
  CreatedAt time.Time
}

// Action
type Action struct {
  ID            uuid.UUID
  Slug          string
  Name          string
  Description   string
  Services      []string // array de nomes
  FunctionPath  *string
  Status        string // active, inactive
  Metadata      *json.RawMessage
  CreatedAt     time.Time
  UpdatedAt     time.Time
}

// Question
type Question struct {
  ID                uuid.UUID
  ActionID          uuid.UUID
  Label             string
  Type              string // radio, text, textarea, file, select
  Options           *json.RawMessage
  IsRequired        bool
  ParentQuestionID  *uuid.UUID
  ParentValue       *string
  ActionKwarg       *string
  IsFirst           bool
  CreatedAt         time.Time
  UpdatedAt         time.Time
}

// ActionExecution
type ActionExecution struct {
  ID          uuid.UUID
  ActionID    uuid.UUID
  Responses   json.RawMessage // {"q1": "CSV", ...}
  Result      json.RawMessage // {"status": "success", ...}
  ExecutedBy  *string
  ExecutedAt  time.Time
  CreatedAt   time.Time
}
```

**Deliverable:** Arquivo compilando sem erros

---

**1.3 Domain Contracts**
Arquivo: `internal/domain/contracts.go`

```go
// HealthChecker qualquer tipo que implemente é válido
type HealthChecker interface {
  Check(ctx context.Context) (*HealthCheckResult, error)
}

type HealthCheckResult struct {
  Status  string // OK, FAILED, UNKNOWN
  Message string
}

// ActionHandler assinatura padrão de handlers
type ActionHandler func(ctx context.Context, responses map[string]interface{}) (*ActionResult, error)

type ActionResult struct {
  Status   string                 // success, error, info
  Message  string
  Data     map[string]interface{} // opcional
  Steps    []string               // opcional (para info)
}

// Repository pattern
type ServiceRepository interface {
  Create(ctx context.Context, s *Service) error
  GetByName(ctx context.Context, name string) (*Service, error)
  GetByID(ctx context.Context, id uuid.UUID) (*Service, error)
  Update(ctx context.Context, s *Service) error
  ListActive(ctx context.Context) ([]*Service, error)
  Delete(ctx context.Context, id uuid.UUID) error
}

type ServiceHealthRepository interface {
  Create(ctx context.Context, sh *ServiceHealth) error
  GetLatestByService(ctx context.Context, serviceID uuid.UUID) (*ServiceHealth, error)
  ListLatestByServices(ctx context.Context, serviceIDs []uuid.UUID) ([]*ServiceHealth, error)
  DeleteOlderThan(ctx context.Context, before time.Time) error
}

type ActionRepository interface {
  Create(ctx context.Context, a *Action) error
  GetBySlug(ctx context.Context, slug string) (*Action, error)
  Update(ctx context.Context, a *Action) error
  ListActive(ctx context.Context) ([]*Action, error)
  Delete(ctx context.Context, id uuid.UUID) error
}

type QuestionRepository interface {
  Create(ctx context.Context, q *Question) error
  ListByActionID(ctx context.Context, actionID uuid.UUID) ([]*Question, error)
  DeleteByActionID(ctx context.Context, actionID uuid.UUID) error
  Update(ctx context.Context, q *Question) error
}

type ActionExecutionRepository interface {
  Create(ctx context.Context, ae *ActionExecution) error
  ListByActionID(ctx context.Context, actionID uuid.UUID, limit int, offset int) ([]*ActionExecution, error)
  Count(ctx context.Context) (int64, error)
}
```

**Deliverable:** Interfaces definidas, pronta para implementação

---

**1.4 Domain Errors**
Arquivo: `internal/domain/errors.go` e `pkg/errors/errors.go`

```go
// internal/domain/errors.go
var (
  ErrServiceNotFound    = errors.New("service not found")
  ErrActionNotFound     = errors.New("action not found")
  ErrQuestionNotFound   = errors.New("question not found")
  ErrInvalidStatus      = errors.New("invalid status")
  ErrDuplicateService   = errors.New("service already exists")
  ErrDuplicateAction    = errors.New("action already exists")
)

// pkg/errors/errors.go (para consumo público)
type APIError struct {
  Code    string
  Message string
  Details string
}
```

**Deliverable:** Estratégia de erros definida

---

**1.5 Migrations SQL**
Diretório: `migrations/`

```sql
-- migrations/000001_init.up.sql

CREATE TABLE IF NOT EXISTS services (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) UNIQUE NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS service_health (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  status VARCHAR(50) NOT NULL,
  message TEXT,
  checked_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug VARCHAR(255) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  services TEXT[],
  function_path VARCHAR(255),
  status VARCHAR(50) DEFAULT 'active',
  metadata JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS questions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action_id UUID NOT NULL REFERENCES actions(id) ON DELETE CASCADE,
  label VARCHAR(255) NOT NULL,
  type VARCHAR(50) NOT NULL,
  options JSONB,
  is_required BOOLEAN DEFAULT false,
  parent_question_id UUID REFERENCES questions(id) ON DELETE CASCADE,
  parent_value VARCHAR(255),
  action_kwarg VARCHAR(255),
  is_first BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS action_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action_id UUID NOT NULL REFERENCES actions(id) ON DELETE CASCADE,
  responses JSONB NOT NULL,
  result JSONB NOT NULL,
  executed_by VARCHAR(255),
  executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices
CREATE INDEX idx_services_is_active ON services(is_active);
CREATE INDEX idx_service_health_service_id_checked_at ON service_health(service_id, checked_at DESC);
CREATE INDEX idx_actions_status ON actions(status);
CREATE INDEX idx_questions_action_id_is_first ON questions(action_id, is_first);
CREATE INDEX idx_action_executions_action_id_executed_at ON action_executions(action_id, executed_at DESC);

-- migrations/000001_init.down.sql
DROP TABLE IF EXISTS action_executions;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS actions;
DROP TABLE IF EXISTS service_health;
DROP TABLE IF EXISTS services;
```

**Deliverable:** Migrations prontas para executar

---

**1.6 SQL Queries (para sqlc)**
Arquivo: `internal/adapters/db/queries.sql`

```sql
-- name: CreateService :one
INSERT INTO services (name, is_active)
VALUES ($1, $2)
RETURNING *;

-- name: GetServiceByName :one
SELECT * FROM services
WHERE name = $1;

-- name: ListActiveServices :many
SELECT * FROM services
WHERE is_active = true
ORDER BY created_at DESC;

-- name: UpdateServiceIsActive :exec
UPDATE services
SET is_active = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreateServiceHealth :one
INSERT INTO service_health (service_id, status, message, checked_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLatestServiceHealth :one
SELECT * FROM service_health
WHERE service_id = $1
ORDER BY checked_at DESC
LIMIT 1;

-- name: DeleteOldServiceHealth :exec
DELETE FROM service_health
WHERE created_at < $1;
```

**Deliverable:** Queries prontas para gerar com sqlc

---

**1.7 go.mod**
Arquivo: `go.mod`

```go
module github.com/caiofariavert/golang_vert_helper

go 1.21

require (
  github.com/gin-gonic/gin v1.9.1
  github.com/google/uuid v1.3.0
  github.com/robfig/cron/v3 v3.0.1
  github.com/gorm.io/gorm v1.25.1
  github.com/gorm.io/driver/postgres v1.5.2
  github.com/golang-migrate/migrate/v4 v4.16.2
  github.com/go-playground/validator/v10 v10.14.0
)
```

**Dependências validadas:**
- `gin`: Framework HTTP (performance, comunidade forte)
- `gorm`: ORM (suporta múltiplos BDs, plugin-ready)
- `postgres`: Driver PostgreSQL para GORM
- Outros: uuid, cron, migrate, validator (padrão)

**Deliverable:** Módulo configurado com dependências validadas

---

#### ✅ Checklist Semana 1
- [ ] Estrutura de pastas criada
- [ ] Entidades definidas
- [ ] Contratos (interfaces) definidos
- [ ] Errors strategy definida
- [ ] Migrations SQL criadas
- [ ] Queries SQL prontas para sqlc
- [ ] go.mod com dependências
- [ ] Projeto compila

---

### **Semana 2: Database + Repositories**

#### Objetivos
- ✅ Pool PostgreSQL conectado
- ✅ Migrations rodando
- ✅ sqlc gerando código type-safe
- ✅ Repositories implementados
- ✅ Testes básicos passando

#### Tarefas

**2.1 PostgreSQL Connection Adapter**
Arquivo: `internal/adapters/db/postgres.go`

```go
type PostgresDB struct {
  *sql.DB
  Queries *Queries // gerado por sqlc
}

func NewPostgresDB(ctx context.Context, dsn string) (*PostgresDB, error) {
  db, err := sql.Open("postgres", dsn)
  if err != nil {
    return nil, err
  }
  
  // Test connection
  if err := db.PingContext(ctx); err != nil {
    return nil, err
  }
  
  // Run migrations
  m, err := migrate.New("file://migrations", dsn)
  if err != nil {
    return nil, err
  }
  
  if err := m.Up(); err != nil && err != migrate.ErrNoChange {
    return nil, err
  }
  
  return &PostgresDB{
    DB:      db,
    Queries: New(db),
  }, nil
}
```

**Deliverable:** Database adapter funcionando

---

**2.2 Generate sqlc**
Arquivo: `sqlc.yaml`

```yaml
version: "2"
project:
  name: "vert_helper"
  package_path: "github.com/caiofariavert/golang_vert_helper/internal/adapters/db"

db:
  driver: "postgres"
  queries: "./internal/adapters/db/queries.sql"
  schema: "./migrations"

out:
  - type: "go"
    dir: "./internal/adapters/db"
```

```bash
# Executar
sqlc generate
```

**Deliverable:** Go code generated from SQL (db.go, models.go, queries.go)

---

**2.3 Repository Implementations**
Arquivo: `internal/repository/service_repo.go`

```go
type ServiceRepo struct {
  db *sql.DB
}

func (r *ServiceRepo) Create(ctx context.Context, s *Service) error {
  _, err := r.db.ExecContext(ctx,
    "INSERT INTO services (id, name, is_active) VALUES ($1, $2, $3)",
    s.ID, s.Name, s.IsActive,
  )
  return err
}

func (r *ServiceRepo) GetByName(ctx context.Context, name string) (*Service, error) {
  var s Service
  err := r.db.QueryRowContext(ctx,
    "SELECT id, name, is_active, created_at, updated_at FROM services WHERE name = $1",
    name,
  ).Scan(&s.ID, &s.Name, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
  return &s, err
}
// ... outros métodos
```

**Deliverable:** Repositories implementados

---

**2.4 Testes de Integração (DB)**
Arquivo: `tests/integration/repository_test.go`

```go
func TestServiceRepositoryCreate(t *testing.T) {
  // Setup testDB (container Docker PostgreSQL)
  db := setupTestDB(t)
  defer db.Close()
  
  repo := repository.NewServiceRepo(db)
  
  s := &domain.Service{
    ID:   uuid.New(),
    Name: "test-service",
    IsActive: true,
  }
  
  err := repo.Create(context.Background(), s)
  if err != nil {
    t.Fatal(err)
  }
  
  retrieved, _ := repo.GetByName(context.Background(), "test-service")
  if retrieved.ID != s.ID {
    t.Error("service not created")
  }
}
```

**Deliverable:** Testes passando

---

#### ✅ Checklist Semana 2
- [ ] PostgreSQL adapter implementado
- [ ] sqlc.yaml criado e rodado
- [ ] Repositories implementados
- [ ] Testes de integração escrevendo
- [ ] DB connection com pooling

---

### **Semana 3: Core Services**

#### Objetivos
- ✅ HealthService orquestrando checks
- ✅ ActionService executando ações
- ✅ SyncService sincronizando BD
- ✅ Health checks built-in (Postgres, S3, Kafka)
- ✅ Testes unitários

#### Tarefas

**3.1 HealthService**
Arquivo: `internal/services/health_service.go`

```go
type HealthService struct {
  serviceRepo ServiceRepository
  healthRepo  ServiceHealthRepository
  checkers    map[string]domain.HealthChecker
  logger      *slog.Logger
}

func (s *HealthService) CheckService(ctx context.Context, serviceName string) (*domain.ServiceHealth, error) {
  // 1. Buscar service no BD
  service, err := s.serviceRepo.GetByName(ctx, serviceName)
  if err != nil {
    return nil, domain.ErrServiceNotFound
  }
  
  // 2. Buscar checker registrado
  checker, ok := s.checkers[serviceName]
  if !ok {
    return nil, fmt.Errorf("checker not registered for %s", serviceName)
  }
  
  // 3. Executar check
  result, err := checker.Check(ctx)
  if err != nil {
    result = &domain.HealthCheckResult{
      Status:  "FAILED",
      Message: err.Error(),
    }
  }
  
  // 4. Persistir resultado
  sh := &domain.ServiceHealth{
    ID:        uuid.New(),
    ServiceID: service.ID,
    Status:    result.Status,
    Message:   &result.Message,
    CheckedAt: time.Now(),
  }
  
  if err := s.healthRepo.Create(ctx, sh); err != nil {
    s.logger.Error("failed to create health record", "error", err)
  }
  
  return sh, nil
}

func (s *HealthService) CheckAllServices(ctx context.Context) error {
  services, err := s.serviceRepo.ListActive(ctx)
  if err != nil {
    return err
  }
  
  var wg sync.WaitGroup
  errs := make(chan error, len(services))
  
  for _, service := range services {
    wg.Add(1)
    go func(svc *domain.Service) {
      defer wg.Done()
      _, err := s.CheckService(ctx, svc.Name)
      if err != nil {
        errs <- err
      }
    }(service)
  }
  
  wg.Wait()
  close(errs)
  
  // Log erros mas não falhe
  for err := range errs {
    s.logger.Error("health check failed", "error", err)
  }
  
  return nil
}

func (s *HealthService) GetLatestStatus(ctx context.Context) (map[string]*HealthStatus, error) {
  services, _ := s.serviceRepo.ListActive(ctx)
  serviceIDs := make([]uuid.UUID, len(services))
  nameMap := make(map[uuid.UUID]string)
  
  for i, svc := range services {
    serviceIDs[i] = svc.ID
    nameMap[svc.ID] = svc.Name
  }
  
  latestHealths, _ := s.healthRepo.ListLatestByServices(ctx, serviceIDs)
  
  result := make(map[string]*HealthStatus)
  for _, h := range latestHealths {
    result[nameMap[h.ServiceID]] = &HealthStatus{
      Status:      h.Status,
      Message:     *h.Message,
      LastUpdated: h.CheckedAt,
    }
  }
  
  return result, nil
}
```

**Deliverable:** HealthService testado

---

**3.2 ActionService**
Arquivo: `internal/services/action_service.go`

```go
type ActionService struct {
  actionRepo    ActionRepository
  questionRepo  QuestionRepository
  executionRepo ActionExecutionRepository
  handlers      map[string]domain.ActionHandler
  logger        *slog.Logger
}

func (s *ActionService) ExecuteAction(ctx context.Context, slug string, responses map[string]interface{}) (*domain.ActionResult, error) {
  // 1. Validar ação existe
  action, err := s.actionRepo.GetBySlug(ctx, slug)
  if err != nil {
    return nil, domain.ErrActionNotFound
  }
  
  // 2. Buscar handler registrado
  handler, ok := s.handlers[slug]
  if !ok {
    return nil, fmt.Errorf("handler not registered for action %s", slug)
  }
  
  // 3. Validar respostas contra perguntas
  if err := s.validateResponses(ctx, action.ID, responses); err != nil {
    return nil, err
  }
  
  // 4. Executar handler
  result, err := handler(ctx, responses)
  if err != nil {
    result = &domain.ActionResult{
      Status:  "error",
      Message: err.Error(),
    }
  }
  
  // 5. Persistir execução
  responsesJSON, _ := json.Marshal(responses)
  resultJSON, _ := json.Marshal(result)
  
  ae := &domain.ActionExecution{
    ID:         uuid.New(),
    ActionID:   action.ID,
    Responses:  responsesJSON,
    Result:     resultJSON,
    ExecutedAt: time.Now(),
  }
  
  if err := s.executionRepo.Create(ctx, ae); err != nil {
    s.logger.Error("failed to record execution", "error", err)
  }
  
  return result, nil
}

func (s *ActionService) validateResponses(ctx context.Context, actionID uuid.UUID, responses map[string]interface{}) error {
  questions, _ := s.questionRepo.ListByActionID(ctx, actionID)
  
  for _, q := range questions {
    if q.IsRequired && responses[q.ID.String()] == nil {
      return fmt.Errorf("required question %s not answered", q.Label)
    }
  }
  
  return nil
}

func (s *ActionService) GetAction(ctx context.Context, slug string) (*ActionDetail, error) {
  action, _ := s.actionRepo.GetBySlug(ctx, slug)
  questions, _ := s.questionRepo.ListByActionID(ctx, action.ID)
  
  return &ActionDetail{
    Action:    action,
    Questions: questions,
  }, nil
}

func (s *ActionService) ListActions(ctx context.Context, filter *ListFilter) ([]*domain.Action, error) {
  actions, _ := s.actionRepo.ListActive(ctx)
  
  // Aplicar filtros (service, search, etc)
  // Ordenar recomendadas primeiro
  
  return actions, nil
}
```

**Deliverable:** ActionService testado

---

**3.3 Health Checks Built-in**
Arquivo: `pkg/health_checks/postgres.go`

```go
type PostgresChecker struct {
  Host     string
  Port     int
  Database string
  User     string
  Password string
}

func (c *PostgresChecker) Check(ctx context.Context) (*domain.HealthCheckResult, error) {
  dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
    c.User, c.Password, c.Host, c.Port, c.Database)
  
  db, err := sql.Open("postgres", dsn)
  if err != nil {
    return &domain.HealthCheckResult{
      Status:  "FAILED",
      Message: "Connection failed: " + err.Error(),
    }, nil
  }
  defer db.Close()
  
  if err := db.PingContext(ctx); err != nil {
    return &domain.HealthCheckResult{
      Status:  "FAILED",
      Message: "Ping failed: " + err.Error(),
    }, nil
  }
  
  return &domain.HealthCheckResult{
    Status:  "OK",
    Message: "PostgreSQL is healthy",
  }, nil
}
```

Similar para S3, Kafka.

**Deliverable:** 3 health checks built-in

---

**3.4 SyncService**
Arquivo: `internal/services/sync_service.go`

```go
type SyncService struct {
  serviceRepo ServiceRepository
  actionRepo  ActionRepository
  questionRepo QuestionRepository
  logger      *slog.Logger
}

// Sincroniza services da configuração com BD
func (s *SyncService) SyncServices(ctx context.Context, configServices map[string]HealthChecker) error {
  // 1. Buscar serviços no BD
  dbServices, _ := s.serviceRepo.ListActive(ctx)
  
  // 2. Para cada service configurado: criar se não existe, ou ativar
  for name := range configServices {
    found := false
    for _, db := range dbServices {
      if db.Name == name {
        found = true
        // Reativar se estava inativo
        if !db.IsActive {
          db.IsActive = true
          s.serviceRepo.Update(ctx, db)
        }
        break
      }
    }
    
    if !found {
      s.serviceRepo.Create(ctx, &domain.Service{
        ID:       uuid.New(),
        Name:     name,
        IsActive: true,
      })
    }
  }
  
  // 3. Desativar serviços removidos da config
  for _, db := range dbServices {
    if _, ok := configServices[db.Name]; !ok && db.IsActive {
      db.IsActive = false
      s.serviceRepo.Update(ctx, db)
    }
  }
  
  return nil
}

// Sincroniza ações registradas com BD
func (s *SyncService) SyncActions(ctx context.Context, registry *ActionRegistry) error {
  // 1. Buscar ações no BD
  dbActions, _ := s.actionRepo.ListActive(ctx)
  
  // 2. Para cada ação registrada: criar ou atualizar
  for _, regAction := range registry.Actions {
    var dbAction *domain.Action
    
    // Buscar se existe
    for _, db := range dbActions {
      if db.Slug == regAction.Slug {
        dbAction = db
        break
      }
    }
    
    if dbAction == nil {
      // Criar
      dbAction = &domain.Action{
        ID:          uuid.New(),
        Slug:        regAction.Slug,
        Name:        regAction.Name,
        Description: regAction.Description,
        Services:    regAction.Services,
        Status:      "active",
      }
      s.actionRepo.Create(ctx, dbAction)
    } else {
      // Atualizar
      dbAction.Name = regAction.Name
      dbAction.Description = regAction.Description
      dbAction.Services = regAction.Services
      s.actionRepo.Update(ctx, dbAction)
    }
    
    // Sincronizar perguntas
    s.syncQuestions(ctx, dbAction.ID, regAction.Questions)
  }
  
  // 3. Deletar ações orfãs
  for _, db := range dbActions {
    found := false
    for _, reg := range registry.Actions {
      if reg.Slug == db.Slug {
        found = true
        break
      }
    }
    
    if !found {
      s.actionRepo.Delete(ctx, db.ID)
    }
  }
  
  return nil
}

func (s *SyncService) syncQuestions(ctx context.Context, actionID uuid.UUID, questions []*helper.Question) error {
  // Deletar perguntas antigas
  s.questionRepo.DeleteByActionID(ctx, actionID)
  
  // Inserir novas
  for i, q := range questions {
    isFirst := (i == 0)
    dbQ := &domain.Question{
      ID:               uuid.New(),
      ActionID:         actionID,
      Label:            q.Label,
      Type:             q.Type,
      IsRequired:       q.IsRequired,
      ActionKwarg:      q.ActionKwarg,
      IsFirst:          isFirst,
      ParentValue:      q.ParentValue,
    }
    
    s.questionRepo.Create(ctx, dbQ)
  }
  
  return nil
}
```

**Deliverable:** SyncService sincronizando dados

---

#### ✅ Checklist Semana 3
- [ ] HealthService implementado e testado
- [ ] ActionService implementado e testado
- [ ] SyncService implementado
- [ ] 3 health checks built-in
- [ ] Testes unitários com >80% cobertura

---

### **Semana 4: HTTP API + Scheduler**

#### Objetivos
- ✅ HTTP handlers para todas APIs
- ✅ Serializações JSON
- ✅ Middleware (auth, errors)
- ✅ Cron scheduler funcionando
- ✅ Limpeza automática de logs

#### Tarefas

**4.1 HTTP Handlers (Gin)**
Arquivo: `internal/adapters/http/handlers.go`

```go
type Handler struct {
  healthService *HealthService
  actionService *ActionService
}

// GET /api/helper/v1/healthcare
func (h *Handler) Healthcare(c *gin.Context) {
  forceRefresh := c.Query("force_refresh") == "true"
  
  ctx := c.Request.Context()
  
  if forceRefresh {
    h.healthService.CheckAllServices(ctx)
  }
  
  status, _ := h.healthService.GetLatestStatus(ctx)
  c.JSON(http.StatusOK, status)
}

// GET /api/helper/v1/actions
func (h *Handler) ListActions(c *gin.Context) {
  ctx := c.Request.Context()
  filter := &ListFilter{
    Service: c.Query("service"),
    Search:  c.Query("search"),
  }
  
  actions, _ := h.actionService.ListActions(ctx, filter)
  c.JSON(http.StatusOK, actions)
}

// GET /api/helper/v1/actions/:slug
func (h *Handler) GetAction(c *gin.Context) {
  slug := c.Param("slug")
  ctx := c.Request.Context()
  
  action, err := h.actionService.GetAction(ctx, slug)
  if err != nil {
    c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
    return
  }
  
  c.JSON(http.StatusOK, action)
}

// POST /api/helper/v1/actions/:slug/execute
func (h *Handler) ExecuteAction(c *gin.Context) {
  slug := c.Param("slug")
  ctx := c.Request.Context()
  
  var req struct {
    Questions map[string]interface{} `json:"questions"`
  }
  
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  
  result, err := h.actionService.ExecuteAction(ctx, slug, req.Questions)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  
  c.JSON(http.StatusOK, result)
}
```

**Benefício:** Handlers Gin, validação automática, binding nativo

**Deliverable:** Handlers implementados

---

**4.2 HTTP Server Setup (Gin)**
Arquivo: `internal/adapters/http/server.go`

```go
func NewGinRouter(handlers *Handler) *gin.Engine {
  router := gin.Default()
  
  // Middleware
  router.Use(loggingMiddleware())
  router.Use(errorMiddleware())
  router.Use(corsMiddleware())
  
  // Routes
  apiV1 := router.Group("/api/helper/v1")
  {
    apiV1.GET("/healthcare", handlers.Healthcare)
    apiV1.GET("/actions", handlers.ListActions)
    apiV1.GET("/actions/:slug", handlers.GetAction)
    apiV1.POST("/actions/:slug/execute", handlers.ExecuteAction)
    apiV1.GET("/app-health", handlers.AppHealth)
  }
  
  return router
}
```

**Benefício:** Gin pronto para produção, middleware ecosystem maduro
```

**Deliverable:** Server HTTP pronto

---

**4.3 Scheduler (Cron)**
Arquivo: `internal/adapters/scheduler/cron.go`

```go
type CronScheduler struct {
  cron *cron.Cron
  logger *slog.Logger
}

func (s *CronScheduler) RegisterHealthCheck(interval time.Duration, job func(context.Context)) error {
  spec := fmt.Sprintf("*/%d * * * * *", interval.Seconds())
  
  _, err := s.cron.AddFunc(spec, func() {
    ctx := context.Background()
    job(ctx)
  })
  
  return err
}

func (s *CronScheduler) RegisterCleanup(interval time.Duration, retention time.Duration, job func(context.Context, time.Duration)) error {
  spec := fmt.Sprintf("*/%d * * * * *", interval.Seconds())
  
  _, err := s.cron.AddFunc(spec, func() {
    ctx := context.Background()
    job(ctx, retention)
  })
  
  return err
}

func (s *CronScheduler) Start() {
  s.cron.Start()
  s.logger.Info("Scheduler started")
}

func (s *CronScheduler) Stop() {
  <-s.cron.Stop().Done()
  s.logger.Info("Scheduler stopped")
}
```

**Deliverable:** Scheduler funcionando

---

**4.4 App Health (Static JSON)**
Arquivo: `internal/adapters/health/app_health.go`

```go
type AppHealthUpdater struct {
  filePath string
  logger   *slog.Logger
}

func (u *AppHealthUpdater) Update(status string) error {
  data := map[string]interface{}{
    "status":    status,
    "timestamp": time.Now().Format(time.RFC3339),
  }
  
  bytes, _ := json.MarshalIndent(data, "", "  ")
  
  if err := os.WriteFile(u.filePath, bytes, 0644); err != nil {
    u.logger.Error("failed to update health file", "error", err)
    return err
  }
  
  return nil
}
```

**Deliverable:** App health JSON atualizado periodicamente

---

#### ✅ Checklist Semana 4
- [ ] HTTP handlers implementados
- [ ] Server HTTP pronto
- [ ] Scheduler (cron) funcionando
- [ ] App health JSON
- [ ] Testes HTTP (integration)

---

### **Semana 5: Exemplos + Documentação + Testes**

#### Objetivos
- ✅ 3+ exemplos funcionais
- ✅ README completo
- ✅ Guia de integração
- ✅ Testes end-to-end
- ✅ Deploy em Docker

#### Tarefas

**5.1 Exemplo 1: Mínimo**
Arquivo: `cmd/examples/basic/main.go`

(Conforme visto em EXEMPLOS_USO.md)

**5.2 Exemplo 2: Com Scheduler**
Arquivo: `cmd/examples/with_scheduler/main.go`

(Conforme visto em EXEMPLOS_USO.md)

**5.3 Exemplo 3: Com Health Checks Custom**
Arquivo: `cmd/examples/with_custom_checks/main.go`

(Conforme visto em EXEMPLOS_USO.md)

**Deliverable:** 3 exemplos compilam e funcionam

---

**5.4 README.md**

```markdown
# Vert Helper - Go Library

## Instalação

go get github.com/caiofariavert/golang_vert_helper

## Quickstart

[exemplo mínimo de 20 linhas]

## Features

- Health monitoring
- Action catalog
- Conditional forms
- REST APIs
- Scheduler integration

## Documentation

- [Configuração](docs/CONFIGURACAO.md)
- [Actions](docs/ACTIONS.md)
- [Health Checks](docs/HEALTH_CHECKS.md)
- [API](docs/API.md)

## Examples

- [Basic](cmd/examples/basic/)
- [With Scheduler](cmd/examples/with_scheduler/)
- [With Custom Checks](cmd/examples/with_custom_checks/)

## License

MIT
```

**Deliverable:** README atractivo

---

**5.5 Docker**

Dockerfile:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/examples/basic

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl
WORKDIR /app

COPY --from=builder /build/app .

EXPOSE 8080
CMD ["./app"]
```

docker-compose.yml:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: vertdb
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  app:
    build: .
    environment:
      DATABASE_URL: "postgres://postgres:password@postgres:5432/vertdb?sslmode=disable"
    ports:
      - "8080:8080"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

**Deliverable:** Docker pronto para rodar

---

**5.6 Testes End-to-End**

Arquivo: `tests/integration/e2e_test.go`

```go
func TestFullFlow(t *testing.T) {
  // Setup
  db := setupTestDB(t)
  defer db.Close()
  
  cfg := helper.NewConfig().
    WithDatabase(testDSN).
    WithService("postgres", &PostgresChecker{...})
  
  h, _ := helper.New(cfg)
  h.Setup(context.Background())
  
  // Register action
  h.RegisterAction("test-action", &helper.Action{
    Handler: func(ctx context.Context, responses map[string]interface{}) (*helper.ActionResult, error) {
      return &helper.ActionResult{Status: "success"}, nil
    },
  })
  
  // Test API
  req, _ := http.NewRequest("GET", "/api/helper/v1/healthcare/", nil)
  rec := httptest.NewRecorder()
  h.Router().ServeHTTP(rec, req)
  
  if rec.Code != 200 {
    t.Errorf("expected 200, got %d", rec.Code)
  }
}
```

**Deliverable:** E2E tests passando

---

#### ✅ Checklist Semana 5
- [ ] 3+ exemplos funcionais
- [ ] README.md completo
- [ ] Docker pronto
- [ ] E2E tests
- [ ] Documentação completa
- [ ] Package publicado (go get works)

---

## 📊 Timeline Visual

```
Semana 1    Semana 2    Semana 3    Semana 4    Semana 5
|-----------|-----------|-----------|-----------|-----------|
Setup       DB          Services    HTTP+Cron   Docs+Tests
Domain      Repos       Sync        APIs        Examples
Entities    Tests       Handlers    Scheduler   E2E
Migrations              Cleanup     Docker
```

---

## 🎯 Definição de Pronto (Definition of Done)

Para cada tarefa:
- ✅ Código escrito
- ✅ Testes passando
- ✅ Documentado
- ✅ Code reviewed
- ✅ Mergeado em main

Para release V1.0:
- ✅ Todos as tarefas acima feitas
- ✅ README completo
- ✅ 3 exemplos funcionando
- ✅ Docker pronto
- ✅ Cobertura de testes >80%
- ✅ Package publicado (go get)
- ✅ Documentação completa
- ✅ Sem TODOs no código

