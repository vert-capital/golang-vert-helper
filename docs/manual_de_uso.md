# Manual de Uso — Vert Helper Go

Biblioteca Go para monitoramento de saúde de serviços e execução de ações interativas. Integra-se a qualquer aplicação Gin/GORM existente sem impor estrutura ou configuração obrigatória.

---

## Índice

1. [Instalação](#1-instalação)
2. [Inicialização](#2-inicialização)
3. [Registrando Serviços e Health Checks](#3-registrando-serviços-e-health-checks)
4. [Registrando Actions](#4-registrando-actions)
5. [Sincronizando Definições com o Banco](#5-sincronizando-definições-com-o-banco)
6. [Registrando Rotas no Gin](#6-registrando-rotas-no-gin)
7. [Scheduler — Health Checks Periódicos](#7-scheduler--health-checks-periódicos)
8. [Monitoramento de Workers](#8-monitoramento-de-workers)
9. [Callbacks](#9-callbacks)
10. [Referência da API REST](#10-referência-da-api-rest)
11. [Referência da API Go](#11-referência-da-api-go)
12. [Como Rodar Migrations](#12-como-rodar-migrations)
13. [Nginx para Consultar Saude](#13-nginx-para-consultar-saude)

---

## 1. Instalação

```bash
go get github.com/caiofariavert/golang_vert_helper
```

**Pré-requisitos:**
- Go 1.21+
- PostgreSQL (ou outro banco suportado pelo GORM)
- Gin já configurado na aplicação

---

## 2. Inicialização

A biblioteca recebe a conexão GORM que **você já possui** — não cria uma nova.

```go
import (
    "github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

// db é sua conexão *gorm.DB existente
h := helper.New(db)
```

### Opções de configuração

```go
import (
    "log/slog"
    "github.com/caiofariavert/golang_vert_helper/pkg/contracts"
    "github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

h := helper.New(db,
    helper.WithLogger(slog.Default()),

    helper.WithOnHealthCheckFailure(func(ctx context.Context, svc *contracts.Service, result *contracts.HealthCheckResult) error {
        log.Printf("ALERTA: serviço %s está %s", svc.Name, result.Status)
        return nil
    }),

    helper.WithOnActionExecution(func(ctx context.Context, exec *contracts.ActionExecution, result *contracts.ActionResult) error {
        log.Printf("Action executada: status=%s", exec.Status)
        return nil
    }),
)
```

---

## 3. Registrando Serviços e Health Checks

Um **serviço** é qualquer dependência que você quer monitorar. Você implementa a interface `HealthChecker` ou usa os checkers prontos.

### Checkers prontos

Quando você registra um serviço, ele é **automaticamente persistido no banco**.

```go
import healthchecks "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"

// Verifica a conexão com o PostgreSQL via GORM
h.RegisterService("postgres", healthchecks.NewGormPostgresChecker(db))

// Verifica a conexão com S3 (AWS)
h.RegisterService("s3", healthchecks.NewS3Checker(healthchecks.S3Config{
    Region: "sa-east-1",
}))

// Verifica a conexão com MinIO ou LocalStack
h.RegisterService("s3", healthchecks.NewS3Checker(healthchecks.S3Config{
    Region:      "us-east-1",
    IsLocal:     true,
    EndpointURL: "http://localhost:9000",
}))

// Com credenciais estáticas (opcional — se omitidas, o SDK usa variáveis de ambiente,
// IAM role ou ~/.aws/credentials automaticamente)
h.RegisterService("s3", healthchecks.NewS3Checker(healthchecks.S3Config{
    Region:          "sa-east-1",
    AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
}))
```

### Checker customizado

Implemente a interface `contracts.HealthChecker`:

```go
type RedisChecker struct {
    client *redis.Client
}

func (c *RedisChecker) Check(ctx context.Context) (*contracts.HealthCheckResult, error) {
    if err := c.client.Ping(ctx).Err(); err != nil {
        return &contracts.HealthCheckResult{
            Status:    contracts.HealthStatusUnhealthy,
            Message:   err.Error(),
            Timestamp: time.Now(),
        }, nil
    }
    return &contracts.HealthCheckResult{
        Status:    contracts.HealthStatusHealthy,
        Message:   "Redis OK",
        Timestamp: time.Now(),
    }, nil
}

// Registrar
h.RegisterService("redis", &RedisChecker{client: redisClient})
```

### Status disponíveis

| Constante | Valor | Significado |
|-----------|-------|-------------|
| `contracts.HealthStatusHealthy` | `"healthy"` | Serviço funcionando normalmente |
| `contracts.HealthStatusDegraded` | `"degraded"` | Funcionando com degradação |
| `contracts.HealthStatusUnhealthy` | `"unhealthy"` | Serviço com falha |
| `contracts.HealthStatusUnknown` | `"unknown"` | Estado desconhecido |

---

## 4. Registrando Actions

Uma **action** representa uma operação interativa — com formulário de entrada — que pode ser executada pelo usuário.

### Estrutura Recomendada

Para organizar melhor o código, **recomenda-se criar um arquivo separado** para registrar as actions. Por exemplo, `internal/actions/actions.go`.

**Arquivo: `internal/actions/actions.go`**

```go
package actions

import (
	"context"
	"log/slog"

	"github.com/caiofariavert/golang_vert_helper/pkg/contracts"
	"github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

// RegisterActions registra todas as actions da aplicação
func RegisterActions(h *helper.Helper, logger *slog.Logger) error {
	// Defina as questões de cada action
	reiniciarServicoQuestions := []contracts.Question{
		{
			Slug:      "servico",
			InputType: contracts.QuestionInputTypeSelect,
			Label:     "Qual serviço?",
			Required:  true,
			Options:   []string{"api", "worker", "scheduler"},
			Order:     1,
		},
		{
			Slug:      "motivo",
			InputType: contracts.QuestionInputTypeTextarea,
			Label:     "Motivo do reinício",
			Required:  true,
			Order:     2,
		},
	}

	// Registre a action com seu handler
	err := h.RegisterAction(
		"reiniciar-servico",
		"Reiniciar Serviço",
		"Reinicia um serviço específico da aplicação",
		func(ctx context.Context, action *contracts.Action, input map[string]interface{}) (*contracts.ActionResult, error) {
			servico := input["servico"].(string)
			motivo  := input["motivo"].(string)

			logger.Info("Reiniciando serviço", "servico", servico, "motivo", motivo)

			// Sua lógica de reinício aqui
			return &contracts.ActionResult{
				Success: true,
				Message: "Serviço reiniciado com sucesso",
				Data:    map[string]interface{}{"servico": servico},
			}, nil
		},
		reiniciarServicoQuestions,
	)

	if err != nil {
		return err
	}

	// Opcionalmente, vincule a action a serviços específicos
	// Isso permite recomendações quando esses serviços falham
	if err := h.LinkActionToService(ctx, "reiniciar-servico", "kafka"); err != nil {
		logger.Error("falha ao vincular action ao serviço", "error", err)
	}

	return nil
}
```

**No seu `main.go` ou inicialização:**

```go
func main() {
	// ... setup do banco e criação do helper ...

	h := helper.New(db)

	// Registre as actions
	if err := actions.RegisterActions(h, logger); err != nil {
		log.Fatalf("falha ao registrar actions: %v", err)
	}

	// ... resto da aplicação ...
}
```

### Registração Inline (Simples)

Se você preferir registrar actions diretamente sem um arquivo separado:

```go
ctx := context.Background()

// Defina as questões
questions := []contracts.Question{
	{
		Slug:      "tipo",
		InputType: contracts.QuestionInputTypeSelect,
		Label:     "Tipo de cache",
		Required:  true,
		Options:   []string{"redis", "memcached", "local"},
		Order:     1,
	},
}

// Registre a action com suas questões
err := h.RegisterAction(
	"limpar-cache",
	"Limpar Cache",
	"Limpa o cache do sistema",
	func(ctx context.Context, action *contracts.Action, input map[string]interface{}) (*contracts.ActionResult, error) {
		tipo := input["tipo"].(string)
		// ... lógica de limpeza
		return &contracts.ActionResult{Success: true, Message: "Cache " + tipo + " limpo"}, nil
	},
	questions,
)

if err != nil {
	log.Fatalf("falha ao registrar action: %v", err)
}
```

### Questões Condicionais (Parent-Child)

```go
questions := []contracts.Question{
	{
		Slug:      "tipo",
		InputType: contracts.QuestionInputTypeSelect,
		Label:     "Tipo do arquivo",
		Required:  true,
		Options:   []string{"CSV", "JSON"},
		Order:     1,
		Children: []contracts.Question{
			{
				Slug:        "csv-fonte",
				InputType:   contracts.QuestionInputTypeSelect,
				Label:       "Fonte do CSV",
				Required:    true,
				Options:     []string{"upload", "url"},
				ParentValue: "CSV",
				Order:       1,
			},
			{
				Slug:        "workflow-id",
				InputType:   contracts.QuestionInputTypeText,
				Label:       "ID do Workflow",
				Required:    true,
				ParentValue: "JSON",
				Order:       1,
			},
		},
	},
}
```

### Vinculando Actions a Serviços

Uma action pode ser vinculada a um ou mais serviços. Essa vinculação é útil para **recomendações**: quando um serviço falha, a API pode sugerir actions vinculadas a ele.

```go
ctx := context.Background()

// Vincule a action a um serviço
if err := h.LinkActionToService(ctx, "processar-documento", "kafka"); err != nil {
	log.Printf("falha ao vincular action: %v", err)
}

// Você pode vincular a mesma action a múltiplos serviços
if err := h.LinkActionToService(ctx, "processar-documento", "s3"); err != nil {
	log.Printf("falha ao vincular action: %v", err)
}
```

Se você preferir usar o método `Sync`:

```go
import "github.com/caiofariavert/golang_vert_helper/pkg/contracts"

// Usar Sync é opcional - apenas para refresh ou em casos especiais
err := h.Sync(ctx, []contracts.ServiceDefinition{
    {
        Name:        "meu-sistema",
        Description: "Sistema principal",
        Actions: []contracts.ActionDefinition{
            {
                Slug:        "reiniciar-servico",
                Title:       "Reiniciar Serviço",
                Description: "Reinicia um serviço específico",
                Questions: []contracts.QuestionDefinition{
                    {
                        Slug:      "servico",
                        InputType: contracts.QuestionInputTypeSelect,
                        Label:     "Qual serviço?",
                        Required:  true,
                        Options:   []string{"api", "worker", "scheduler"},
                        Order:     1,
                    },
                    {
                        Slug:      "motivo",
                        InputType: contracts.QuestionInputTypeTextarea,
                        Label:     "Motivo do reinício",
                        Required:  true,
                        Order:     2,
                    },
                },
            },
        },
    },
})
```

---

## 6. Registrando Rotas no Gin

```go
router := gin.Default()

// Sem middleware
h.RegisterRoutes(router, db, nil)

// Com middleware (ex: autenticação JWT)
authMiddleware := func(c *gin.Context) {
    token := c.GetHeader("Authorization")
    if token == "" {
        c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
        return
    }
    c.Next()
}
h.RegisterRoutes(router, db, &authMiddleware)

router.Run(":8080")
```

Todas as rotas são registradas sob o prefixo `/api/helper/v1/`.

---

## 7. Scheduler — Health Checks Periódicos

O scheduler executa health checks automaticamente em background usando cron.

```go
// Configuração padrão: a cada 10 minutos
scheduler := helper.NewScheduler(h, helper.DefaultSchedulerConfig())
defer scheduler.Stop()
```

```go
// Configuração customizada
scheduler := helper.NewScheduler(h, helper.SchedulerConfig{
    HealthCheckCron: "*/5 * * * *", // a cada 5 minutos
})
defer scheduler.Stop()
```

### Expressões cron úteis

| Expressão | Frequência |
|-----------|------------|
| `"*/5 * * * *"` | A cada 5 minutos |
| `"*/10 * * * *"` | A cada 10 minutos (padrão) |
| `"0 * * * *"` | A cada hora |
| `"0 9 * * *"` | Todo dia às 9h |

---

## 8. Monitoramento de Workers

O `WorkerPool` permite registrar goroutines/jobs e monitorar seu estado em tempo real.

### Setup

```go
import healthchecks "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"

pool := healthchecks.NewWorkerPool()

h := helper.New(db,
    helper.WithWorkerPool(pool), // expõe automaticamente em /api/helper/v1/workers/
)
```

### Usando no seu worker

```go
func StartKafkaConsumer(pool *healthchecks.WorkerPool) {
    pool.Register("kafka-consumer-1", "Kafka Consumer Principal")

    go func() {
        for msg := range messages {
            pool.UpdateStatus("kafka-consumer-1", healthchecks.WorkerProcessing, "")

            if err := process(msg); err != nil {
                pool.UpdateStatus("kafka-consumer-1", healthchecks.WorkerFailed, err.Error())
                continue
            }

            pool.UpdateStatus("kafka-consumer-1", healthchecks.WorkerIdle, "")
        }
    }()
}
```

### Status de worker disponíveis

| Constante | Valor | Quando usar |
|-----------|-------|-------------|
| `WorkerRunning` | `"running"` | Worker iniciado, aguardando mensagens |
| `WorkerIdle` | `"idle"` | Aguardando trabalho |
| `WorkerProcessing` | `"processing"` | Processando uma mensagem |
| `WorkerFailed` | `"failed"` | Erro na última execução |
| `WorkerPaused` | `"paused"` | Pausado intencionalmente |
| `WorkerBackoff` | `"backoff"` | Em espera após falha |

---

## 9. Callbacks

### Falha em health check

```go
h := helper.New(db,
    helper.WithOnHealthCheckFailure(func(ctx context.Context, svc *contracts.Service, result *contracts.HealthCheckResult) error {
        // Notificar Slack, PagerDuty, etc.
        notifySlack(fmt.Sprintf("⚠️ %s está %s: %s", svc.Name, result.Status, result.Message))
        return nil
    }),
)
```

### Execução de action

```go
h := helper.New(db,
    helper.WithOnActionExecution(func(ctx context.Context, exec *contracts.ActionExecution, result *contracts.ActionResult) error {
        // Auditoria, logs, notificações
        audit.Log("action_executed", exec.ActionID, exec.Status)
        return nil
    }),
)
```

---

## 10. Referência da API REST

Todas as rotas são prefixadas com `/api/helper/v1`.

### Health Checks

#### `GET /api/helper/v1/healthcare/`
Retorna o status geral de todos os serviços monitorados.

**Query param opcional:** `force_refresh=true` para executar os checks imediatamente.

**Response 200:**
```json
{
  "postgres": {
    "status": "healthy",
    "message": "PostgreSQL connection is healthy",
    "last_updated": "2026-07-21T10:00:00Z"
  },
  "s3": {
    "status": "unhealthy",
    "message": "Connection timeout",
    "last_updated": "2026-07-21T10:05:00Z"
  },
  "kafka": {
    "status": "unknown",
    "message": "Service not configured",
    "last_updated": null
  }
}
```

#### `GET /api/helper/v1/healthcare/:name`
Retorna o último status registrado de um serviço específico.

**Query param opcional:** `force_refresh=true` para executar o check imediatamente.

**Response 200:**
```json
{
  "id": "uuid",
  "service_id": "uuid",
  "status": "healthy",
  "message": "OK",
  "checked_at": "2026-07-21T10:00:00Z"
}
```

**Response 404:** Serviço não encontrado.

---

### Actions

#### `GET /api/helper/v1/actions/?service_id=<uuid>`
Lista todas as actions de um serviço.

**Query param obrigatório:** `service_id`

**Response 200:**
```json
[
  {
    "id": "uuid",
    "service_id": "uuid",
    "slug": "reiniciar-servico",
    "title": "Reiniciar Serviço",
    "description": "...",
    "active": true,
    "questions": [...]
  }
]
```

#### `GET /api/helper/v1/actions/:slug`
Retorna o detalhe de uma action, incluindo suas questões.

**Response 200:**
```json
{
  "id": "uuid",
  "slug": "reiniciar-servico",
  "title": "Reiniciar Serviço",
  "questions": [
    {
      "slug": "servico",
      "input_type": "select",
      "label": "Qual serviço?",
      "required": true,
      "options": "[\"api\",\"worker\"]",
      "order": 1,
      "children": []
    }
  ]
}
```

#### `POST /api/helper/v1/actions/:slug/execute`
Executa uma action com as respostas do formulário.

**Request body:**
```json
{
  "servico": "api",
  "motivo": "Alta de memória detectada"
}
```

**Response 200:**
```json
{
  "success": true,
  "message": "Serviço reiniciado com sucesso",
  "data": {
    "servico": "api"
  }
}
```

**Response 422:** Campos obrigatórios faltando.  
**Response 404:** Action não encontrada.

---

### Workers

#### `GET /api/helper/v1/workers/`
Lista todos os workers registrados no WorkerPool.

**Response 200:**
```json
[
  {
    "id": "kafka-consumer-1",
    "name": "Kafka Consumer Principal",
    "status": "idle",
    "last_check": "2026-07-21T10:04:55Z",
    "last_error": "",
    "processed_count": 1543,
    "failed_count": 2
  }
]
```

#### `GET /api/helper/v1/workers/:id`
Retorna o detalhe de um worker específico.

**Response 200:** Mesmo formato de um item da lista acima.  
**Response 404:** Worker não encontrado.

---

## 11. Referência da API Go

### `helper.New(db *gorm.DB, opts ...Option) *Helper`
Cria uma instância do Helper.

### `h.RegisterService(name string, checker contracts.HealthChecker) error`
Registra um health checker para um serviço e **sincroniza automaticamente** no banco.

**Retorna um erro se a sincronização falhar.**

### `h.RegisterAction(slug, title, description string, handler contracts.ActionHandler, questions []contracts.Question) error`
Registra uma action com suas questões e **sincroniza automaticamente** no banco.

**Parâmetros:**
- `slug`: Identificador único da action (ex: "limpar-cache")
- `title`: Título exibível (ex: "Limpar Cache")
- `description`: Descrição da action
- `handler`: Função que executa a action
- `questions`: Array de questões do formulário (pode ser vazio)

**Retorna um erro se a sincronização falhar.**

### `h.LinkActionToService(ctx context.Context, actionSlug, serviceName string) error`
Vincula uma action a um serviço. Uma action pode ser vinculada a múltiplos serviços.

**Características:**
- Não cria duplicatas (idempotente)
- Retorna `nil` se o vínculo já existe

### `h.RegisterRoutes(router *gin.Engine, db *gorm.DB, middleware *gin.HandlerFunc)`
Registra todas as rotas no router Gin do cliente. Passa `nil` como middleware para sem autenticação.

### `h.Sync(ctx context.Context, defs []services.ServiceDefinition) error`
**Opcional.** Sincroniza definições de serviços e actions com o banco. Útil para refresh de dados.

**Nota:** Normalmente não é necessário pois `RegisterService` e `RegisterAction` já sincronizam automaticamente.

### `h.AutoMigrate() error`
Executa migrations de schema via GORM para todos os modelos do helper (incluindo a nova tabela `gohelper_action_services`) em uma única chamada.

### `h.CheckService(ctx, name) (*contracts.HealthCheckResult, error)`
Executa o health check de um serviço manualmente.

### `h.CheckAll(ctx) map[string]*contracts.HealthCheckResult`
Executa health checks de todos os serviços registrados.

### `h.ExecuteAction(ctx, slug, input) (*contracts.ActionResult, error)`
Executa uma action programaticamente (sem HTTP).

### `helper.NewScheduler(h *Helper, cfg SchedulerConfig) *Scheduler`
Cria e inicia o scheduler de health checks periódicos.

### `helper.DefaultSchedulerConfig() SchedulerConfig`
Retorna a configuração padrão (health check a cada 10 minutos).

---

## 12. Como Rodar Migrations

O schema do helper deve ser aplicado via GORM `AutoMigrate`, em lote e de forma automatica.
Nao rode alteracoes manualmente tabela por tabela.

### Via codigo com GORM (recomendado)

Use o helper para rodar `AutoMigrate` na inicializacao da aplicacao:

```go
h := helper.New(db)

if err := h.AutoMigrate(); err != nil {
    log.Fatalf("falha ao executar migrations: %v", err)
}
```

Esse metodo aplica automaticamente o schema de todos os modelos do pacote:
- `gohelper_services` - serviços monitorados
- `gohelper_service_health` - histórico de health checks
- `gohelper_actions` - ações disponíveis
- `gohelper_action_services` - vinculação many-to-many entre actions e services
- `gohelper_questions` - questões dos formulários
- `gohelper_action_executions` - auditoria de execuções
- `gohelper_workers` - workers monitorados
- `gohelper_worker_snapshots` - histórico de workers

Se voce inicializa via adapter interno, o processo ja e automatico:

```go
initializer, err := adapters.NewApplicationInitializer(config)
if err != nil {
    return err
}
defer initializer.Close()
```

Nesse fluxo, `NewApplicationInitializer` chama `AutoMigrate()` internamente.

---

## 13. Nginx para Consultar Saude

O endpoint publico de saude da aplicacao e servido pelo Nginx em:

- `GET /api/helper/v1/app-health/`

Esse endpoint retorna o arquivo `/app/health.json`, atualizado pelo script `health_check.sh`.
O script consulta internamente `GET /api/helper/v1/healthcare/` e grava:

- `status`: `stable` quando o helper responde
- `status`: `failed` quando o helper nao responde

### Fluxo de funcionamento

1. A aplicacao sobe e expõe o helper em `http://127.0.0.1:8006`.
2. O script `health_check.sh` roda em background no container (inicial e periodico).
3. O script gera/atualiza `/app/health.json`.
4. O Nginx serve esse arquivo em `/api/helper/v1/app-health/`.

### Configuracao Nginx no projeto

- Desenvolvimento: `docker_assets/nginx.dev.conf` (porta `8005`)
- Producao: `docker_assets/nginx.prd.conf` (porta `8080`)

Nos dois cenarios:

- `/api/helper/v1/app-health` redireciona para `/api/helper/v1/app-health/`
- `/api/helper/v1/app-health/` usa `try_files /health.json =503`
- Demais rotas sao encaminhadas para `http://127.0.0.1:8006`

### Como aplicar no container

Os Dockerfiles ja aplicam a configuracao automaticamente:

- `Dockerfile-dev` copia `docker_assets/nginx.dev.conf` para `/etc/nginx/conf.d/default.conf`
- `Dockerfile-prod` copia `docker_assets/nginx.prd.conf` para `/etc/nginx/conf.d/default.conf`
- Ambos copiam e executam `/app/health_check.sh`

### Como consultar saude

Em desenvolvimento:

```bash
curl -fsS http://localhost:8005/api/helper/v1/app-health/
```

Em producao:

```bash
curl -fsS http://localhost:8080/api/helper/v1/app-health/
```

Exemplo de resposta:

```json
{
    "status": "stable",
    "timestamp": "2026-07-22T10:30:00-03:00"
}
```

Se `health.json` ainda nao existir, o Nginx responde `503` ate o primeiro ciclo do `health_check.sh`.

---

## Exemplo completo de integração

### Arquivo: `internal/actions/actions.go`

```go
package actions

import (
	"context"
	"log/slog"

	"github.com/caiofariavert/golang_vert_helper/pkg/contracts"
	"github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

// RegisterActions registra todas as actions da aplicação
func RegisterActions(h *helper.Helper, ctx context.Context, logger *slog.Logger) error {
	// Definir questões
	cleanCacheQuestions := []contracts.Question{
		{
			Slug:      "tipo",
			InputType: contracts.QuestionInputTypeSelect,
			Label:     "Tipo de cache",
			Required:  true,
			Options:   []string{"redis", "memcached", "local"},
			Order:     1,
		},
	}

	// Registrar action
	if err := h.RegisterAction(
		"limpar-cache",
		"Limpar Cache",
		"Limpa o cache do sistema",
		func(ctx context.Context, action *contracts.Action, input map[string]interface{}) (*contracts.ActionResult, error) {
			tipo := input["tipo"].(string)
			logger.Info("Limpando cache", "tipo", tipo)
			return &contracts.ActionResult{
				Success: true,
				Message: "Cache " + tipo + " limpo",
			}, nil
		},
		cleanCacheQuestions,
	); err != nil {
		return err
	}

	// Vincular a ações a serviços (opcional)
	if err := h.LinkActionToService(ctx, "limpar-cache", "redis"); err != nil {
		logger.Error("falha ao vincular action", "error", err)
	}

	return nil
}
```

### Arquivo: `cmd/main.go`

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"myapp/internal/actions"
	"github.com/caiofariavert/golang_vert_helper/pkg/contracts"
	healthchecks "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
	"github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

func main() {
	logger := slog.Default()

	// Sua conexão GORM existente
	db, _ := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})

	// WorkerPool para monitorar goroutines (opcional)
	pool := healthchecks.NewWorkerPool()

	// Inicializar helper
	h := helper.New(db,
		helper.WithLogger(logger),
		helper.WithWorkerPool(pool),
		helper.WithOnHealthCheckFailure(func(ctx context.Context, svc *contracts.Service, result *contracts.HealthCheckResult) error {
			logger.Error("serviço com falha", "service", svc.Name, "status", result.Status)
			return nil
		}),
	)

	// Executar migrations
	if err := h.AutoMigrate(); err != nil {
		logger.Fatalf("falha ao executar migrations: %v", err)
	}

	// Registrar health checkers
	// Isso sincroniza automaticamente no banco
	if err := h.RegisterService("postgres", healthchecks.NewGormPostgresChecker(db)); err != nil {
		logger.Fatalf("falha ao registrar postgres checker: %v", err)
	}

	if err := h.RegisterService("s3", healthchecks.NewS3Checker(healthchecks.S3Config{
		Region: "sa-east-1",
	})); err != nil {
		logger.Fatalf("falha ao registrar s3 checker: %v", err)
	}

	// Registrar actions
	ctx := context.Background()
	if err := actions.RegisterActions(h, ctx, logger); err != nil {
		logger.Fatalf("falha ao registrar actions: %v", err)
	}

	// Iniciar scheduler (health check periódico)
	scheduler := helper.NewScheduler(h, helper.DefaultSchedulerConfig())
	defer scheduler.Stop()

	// Seu router Gin existente
	router := gin.Default()

	// Suas próprias rotas
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"pong": true})
	})

	// Registrar rotas do helper
	h.RegisterRoutes(router, db, nil) // sem middleware

	// Registrar worker (opcional)
	pool.Register("worker-principal", "Worker Principal")
	go runWorker(pool)

	// Iniciar servidor
	router.Run(":8080")
}

func runWorker(pool *healthchecks.WorkerPool) {
	pool.UpdateStatus("worker-principal", healthchecks.WorkerRunning, "")
	for {
		pool.UpdateStatus("worker-principal", healthchecks.WorkerProcessing, "")
		// ... processar tarefa
		pool.UpdateStatus("worker-principal", healthchecks.WorkerIdle, "")
	}
}
```
