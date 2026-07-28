# Plano de Desenvolvimento - Vert Helper Go Library

## 📋 Visão Geral

Esta é uma análise e plano de implementação da biblioteca Vert Helper em Go, baseado nas especificações técnicas agnósticas e Django.

**Objetivo:** Criar uma biblioteca Go que funcione como um "plugin" fácil de integrar em projetos existentes, oferecendo:
- ✅ Monitoramento de saúde de serviços externos
- ✅ Catálogo e execução de ações operacionais
- ✅ Formulários condicionais para entrada de dados
- ✅ APIs REST para consumo
- ✅ Agendamento de rotinas periódicas

---

## 1. Análise Comparativa: Django vs Go

### 1.1 Configuração de Serviços

#### Django
```python
VERT_HELPER = {
    "SERVICES": {
        "postgres": {
            "function": "module.check_postgres",
            "context": {...}
        }
    }
}
```

#### Go - Decisão: **Config-driven + Type-safe**
```go
// config.yaml
services:
  postgres:
    enabled: true
    function: "checks.CheckPostgres"  // reflexão
    context:
      host: localhost
      port: 5432
```

Ou (mais idiomático Go):
```go
helper.RegisterService("postgres", 
  ServiceConfig{
    Function: checks.CheckPostgres,
    Context: map[string]interface{}{...}
  })
```

**Recomendação:** Usar **builder pattern** + **struct config**. Go é type-safe, então preferir:
```go
cfg := helper.NewConfig().
  WithService("postgres", helper.PostgresChecker{
    Host: "localhost",
    Port: 5432,
  }).
  WithService("s3", helper.S3Checker{
    Bucket: "my-bucket",
  })
```

---

### 1.2 Registro de Ações

#### Django
```python
@helper_action(
    slug="execute-without-kafka",
    services=["S3", "KAFKA"],
    questions=[...]
)
def execute_without_kafka(responses):
    return {"status": "success", "data": {...}}
```

#### Go - Decisão: **Explicit Registry Pattern**

Opção A - Registry function (simples):
```go
helper.RegisterAction("execute-without-kafka",
  &Action{
    Name: "Executar sem Kafka",
    Services: []string{"S3", "KAFKA"},
    Handler: func(ctx context.Context, responses map[string]interface{}) (*ActionResult, error) {
      // implementação
    },
    Questions: []*Question{...},
  })
```

Opção B - Struct fields com init (mais Go-like):
```go
type MyActions struct {
  ExecuteWithoutKafka *Action
}

func (a *MyActions) Init() error {
  a.ExecuteWithoutKafka = &Action{
    Slug: "execute-without-kafka",
    // ...
  }
  return helper.RegisterAction(a.ExecuteWithoutKafka)
}
```

**Recomendação:** Opção A (simples) + helper package com exemplos de Opção B

---

### 1.3 Modelo de Dados

| Entidade | Django | Go | Status |
|----------|--------|----|----|
| Service | ORM automático | Struct + sqlc | Mesmo |
| ServiceHealth | ORM automático | Struct + sqlc | Mesmo |
| Action | ORM automático | Struct + sqlc | Mesmo |
| Question | ORM automático + relacionamento recursivo | Struct + sqlc | Mesmo |
| ActionExecution | ORM automático | Struct + sqlc | Mesmo |

**Decisão:** Usar `sqlc` para type-safe SQL com migrations do `golang-migrate`.

---

## 2. Decisões de Design

### 2.1 Framework HTTP

**Opções:**
- `net/http` puro (zero dependências)
- `gin` (performance + simples)
- `echo` (minimalista + robusto)
- `fiber` (inspirado em Express)

**Recomendação:** `gin` (validado pelo usuário) ✅

**Justificativa:**
- Performance excelente
- API simples e intuitiva
- Middleware ecosystem maduro
- Fácil integração com projetos existentes
- Comunidade forte em Go

---

### 2.2 Banco de Dados

**Opções:**
- GORM (ORM completo, maduro, com hooks)
- sqlc (type-safe, zero-runtime overhead)
- database/sql puro

**Recomendação:** `GORM` + `golang-migrate` (validado pelo usuário) ✅

**Justificativa:**
- ORM maduro e bem mantido
- Suporta múltiplos bancos (PostgreSQL, MySQL, SQLite, etc)
- Hooks para business logic
- Migrations automáticas opcionais
- Plugin architecture já pronta
- Usuários podem estender com outras conexões facilmente

**Suporte inicial:** PostgreSQL (conforme spec), arquitetura permite múltiplos BDs

---

### 2.3 Scheduler

**Opções:**
- `robfig/cron` (mencionado na spec, maduro)
- `gocron` (simpler API)
- goroutine + `time.Ticker` (built-in)

**Recomendação:** `robfig/cron` v3

**Justificativa:**
- Mencionado na spec agnóstica
- Mature ecosystem
- Padrão de cron familiar
- Fácil agendamento

---

### 2.4 Logging

**Opções:**
- `slog` (stdlib Go 1.21+)
- `zap` (estruturado, performance)
- `logrus` (popular, maduro)

**Recomendação:** `slog` (stdlib) com fallback para `zap`

**Justificativa:**
- Padrão Go 1.21+
- Estruturado nativo
- Zero dependências opcionais

---

### 2.5 Validação

**Recomendação:** `github.com/go-playground/validator/v10`

Simples, maduro, bem mantido.

---

## 3. Estrutura de Pacotes

```
golang_vert_helper/
├── cmd/
│   └── examples/
│       ├── basic/
│       │   └── main.go           # Exemplo mínimo
│       ├── with_scheduler/
│       │   └── main.go           # Com agendamento
│       └── with_custom_checks/
│           └── main.go           # Com health checks custom
├── internal/
│   ├── domain/
│   │   ├── entities.go           # Service, Action, Question, etc
│   │   ├── contracts.go          # Interfaces (HealthChecker, ActionHandler)
│   │   └── errors.go             # Domain errors
│   ├── adapters/
│   │   ├── db/
│   │   │   ├── postgres.go       # Conexão PostgreSQL
│   │   │   └── queries.sql.go    # sqlc generated
│   │   ├── scheduler/
│   │   │   └── cron.go           # robfig/cron adapter
│   │   └── http/
│   │       ├── server.go         # net/http server
│   │       ├── handlers.go       # HTTP handlers
│   │       └── middleware.go     # Middleware
│   ├── services/
│   │   ├── health_service.go     # Orquestração de health checks
│   │   ├── action_service.go     # Orquestração de ações
│   │   └── sync_service.go       # Sync de actions/services
│   └── repository/
│       ├── service_repo.go       # Repository pattern
│       ├── action_repo.go
│       ├── question_repo.go
│       └── execution_repo.go
├── pkg/
│   ├── helper/
│   │   ├── helper.go             # Public API main
│   │   ├── config.go             # Config builder
│   │   ├── registry.go           # Action registry
│   │   └── types.go              # Tipos públicos
│   ├── health_checks/
│   │   ├── postgres.go           # Check Postgres (built-in)
│   │   ├── s3.go                 # Check S3 (built-in)
│   │   └── kafka.go              # Check Kafka (built-in)
│   └── errors/
│       └── errors.go             # Public errors
├── migrations/
│   ├── 001_init.up.sql
│   ├── 001_init.down.sql
│   └── ...
├── scripts/
│   ├── setup.sh                  # Setup dev environment
│   └── migrate.sh                # Run migrations
├── tests/
│   ├── integration/
│   └── unit/
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── ARQUITETURA.md               # Decisões técnicas (este doc)
└── docs/
    ├── CONFIGURACAO.md          # Guia de config
    ├── ACTIONS.md               # Como registrar ações
    ├── HEALTH_CHECKS.md         # Como implementar health checks
    └── API.md                   # Documentação de API
```

---

## 4. Interface Pública (API ao usuário)

### 4.1 Inicialização Mínima

```go
package main

import (
  "context"
  "log"
  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

func main() {
  // 1. Configurar
  cfg := helper.NewConfig().
    WithDatabase("postgres://user:pass@localhost/db").
    WithService("postgres", helper.PostgresChecker{
      Host: "localhost",
      Port: 5432,
      Database: "mydb",
    }).
    WithService("s3", helper.S3Checker{
      Bucket: "my-bucket",
    })

  // 2. Instanciar
  h, err := helper.New(cfg)
  if err != nil {
    log.Fatal(err)
  }

  // 3. Registrar ações
  h.RegisterAction("my-action", &helper.Action{
    Name: "Minha Ação",
    Handler: myActionHandler,
    Questions: [...],
  })

  // 4. Setup (sync services/actions, scheduler)
  if err := h.Setup(context.Background()); err != nil {
    log.Fatal(err)
  }

  // 5. Servir APIs com Gin
  router := h.GinRouter() // retorna *gin.Engine
  router.Run(":8080")
}
```

### 4.2 Estruturas Públicas

```go
type Config struct {
  // Banco de dados
  DatabaseURL string
  
  // Scheduler
  SchedulerEnabled bool
  HealthCheckInterval time.Duration // default: 10m
  
  // Services pré-registrados
  Services map[string]HealthChecker
  
  // Logging
  Logger *slog.Logger
  
  // Permissões
  PermissionClass PermissionChecker // default: AllowAll
}

type Action struct {
  Slug string
  Name string
  Description string
  Services []string
  Handler ActionHandler
  Questions []*Question
}

type Question struct {
  ID string
  Label string
  Type string // "radio", "text", "textarea", "file", "select"
  Options []string
  IsRequired bool
  ParentQuestionID *string
  ParentValue *string
  ActionKwarg *string
}

type ActionHandler func(ctx context.Context, responses map[string]interface{}) (*ActionResult, error)

type ActionResult struct {
  Status string // "success", "error", "info"
  Message string
  Data map[string]interface{}
  Steps []string
}

type HealthChecker interface {
  Check(ctx context.Context) (*HealthCheckResult, error)
}

type HealthCheckResult struct {
  Status string // "OK", "FAILED", "UNKNOWN"
  Message string
}
```

---

## 5. Contratos Internos

### 5.1 Health Check

Qualquer tipo que implemente `HealthChecker` é válido:

```go
type PostgresChecker struct {
  Host string
  Port int
  Database string
  User string
  Password string
}

func (c *PostgresChecker) Check(ctx context.Context) (*HealthCheckResult, error) {
  db, err := sql.Open("postgres", c.DSN())
  if err != nil {
    return &HealthCheckResult{Status: "FAILED", Message: err.Error()}, nil
  }
  defer db.Close()
  
  if err := db.PingContext(ctx); err != nil {
    return &HealthCheckResult{Status: "FAILED", Message: err.Error()}, nil
  }
  
  return &HealthCheckResult{Status: "OK"}, nil
}
```

### 5.2 Action Handler

Signature padrão:
```go
func MyAction(ctx context.Context, responses map[string]interface{}) (*ActionResult, error) {
  // Processar responses (respostas do formulário)
  // Retornar resultado estruturado
}
```

---

## 6. APIs HTTP Expostas

```
GET    /api/helper/v1/healthcare/
GET    /api/helper/v1/actions/
GET    /api/helper/v1/actions/:slug
POST   /api/helper/v1/actions/:slug/execute
GET    /api/helper/v1/app-health/   (arquivo estático)
```

**Nota:** APIs disponíveis como Gin handlers, fácil integração com projetos existentes

---

## 7. Migrations SQL

Schema inicial (PostgreSQL):

```sql
-- Services
CREATE TABLE services (
  id UUID PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Service Health
CREATE TABLE service_health (
  id UUID PRIMARY KEY,
  service_id UUID NOT NULL REFERENCES services(id),
  status VARCHAR(50) NOT NULL,
  message TEXT,
  checked_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Actions
CREATE TABLE actions (
  id UUID PRIMARY KEY,
  slug VARCHAR(255) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  services TEXT[], -- Array de nomes
  function_path VARCHAR(255),
  status VARCHAR(50) DEFAULT 'active',
  metadata JSONB,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Questions
CREATE TABLE questions (
  id UUID PRIMARY KEY,
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

-- Action Executions
CREATE TABLE action_executions (
  id UUID PRIMARY KEY,
  action_id UUID NOT NULL REFERENCES actions(id),
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
```

---

## 8. Dependências Go

```toml
# go.mod
require (
  github.com/gin-gonic/gin v1.9.x
  github.com/google/uuid v1.3.x
  github.com/robfig/cron/v3 v3.0.x
  github.com/gorm.io/gorm v1.25.x
  github.com/gorm.io/driver/postgres v1.5.x
  github.com/golang-migrate/migrate/v4 v4.x.x
  github.com/go-playground/validator/v10 v10.x.x
  golang.org/x/sync v0.x.x
)
```

**Justificativa de dependências:**
- `gin`: Framework HTTP validado
- `gorm`: ORM validado, suporta múltiplos BDs
- `uuid`: padrão (pequeno)
- `cron`: especificação recomenda
- `postgres`: Driver PostgreSQL (GORM)
- `golang-migrate`: migrações
- `validator`: validação
- `sync`: primitivas concorrência

Total: **6 dependências** (vs Django com ~50+)

---

## 9. Workflow de Desenvolvimento

### Fase 1: Setup e Domain (Semana 1)
- [x] Estrutura de pastas
- [x] Models/entities
- [x] Contracts (interfaces)
- [x] Migrations
- [x] Database setup

### Fase 2: Core Services (Semana 2)
- [ ] Health check service
- [ ] Action registry + executor
- [ ] Sync service
- [ ] Health checks built-in (Postgres, S3, Kafka)

### Fase 3: HTTP API (Semana 3)
- [ ] Handlers HTTP
- [ ] Serialization (JSON)
- [ ] Middleware (auth, errors)
- [ ] Documentação OpenAPI

### Fase 4: Scheduler + Integração (Semana 4)
- [ ] Cron adapter
- [ ] Health check scheduler
- [ ] Cleanup job
- [ ] Docker + health.json

### Fase 5: Exemplos e Docs (Semana 5)
- [ ] 3+ exemplos funcionais
- [ ] README detalhado
- [ ] Guia de integração
- [ ] Testes automatizados

---

## 10. Definições de Sucesso

✅ **V1 Mínima (MVP):**
- Configuração simples em ~20 linhas de código
- Registrar ação em ~15 linhas
- Health checks funcionando
- APIs RESTful conforme spec
- Testes com cobertura >80%

✅ **Características:**
- Zero dependências obrigatórias (exceto driver PostgreSQL)
- Setup automático de schema
- Exemplos funcionais
- Documentação completa

---

## 11. Próximos Passos

1. ✅ **Validar este plano** (discussão com time)
2. **Criar estrutura de pastas** + setup Go module
3. **Definir tipos Go** para domínio
4. **Implementar migrations**
5. **Começar com Health Service**

---

## 📝 Notas Importantes

- Go força mais explicitness que Python (sem decorators)
- Registry pattern é idiomático para Go
- Preferir `net/http` puro = menos vendor lock-in
- `sqlc` garante type-safety que ORM pode não dar
- Estrutura pronta para escalar (domain-driven)

