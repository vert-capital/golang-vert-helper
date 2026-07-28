# Exemplos de Uso - Vert Helper Go

Este documento mostra como seria usar a biblioteca após implementação, para validar as decisões de design.

---

## Exemplo 1: Integração Mínima (MVP)

```go
package main

import (
  "context"
  "log"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  ctx := context.Background()

  // 1️⃣ Configurar
  cfg := helper.NewConfig().
    WithDatabase("postgres://user:pass@localhost:5432/vertdb").
    WithService("postgres", &health_checks.PostgresChecker{
      Host:     "localhost",
      Port:     5432,
      Database: "vertdb",
      User:     "postgres",
      Password: "password",
    }).
    WithService("s3", &health_checks.S3Checker{
      Bucket: "my-bucket",
      Region: "us-east-1",
    })

  // 2️⃣ Instanciar helper
  h, err := helper.New(cfg)
  if err != nil {
    log.Fatal(err)
  }

  // 3️⃣ Registrar uma ação
  h.RegisterAction("execute-without-kafka", &helper.Action{
    Name:        "Executar sem Kafka",
    Description: "Executa operação sem dependência do Kafka",
    Services:    []string{"S3", "KAFKA"},
    Handler:     executeWithoutKafkaHandler,
    Questions: []*helper.Question{
      {
        ID:         "q1",
        Label:      "Qual o tipo do arquivo?",
        Type:       "radio",
        Options:    []string{"CSV", "JSON"},
        IsRequired: true,
        IsFirst:    true,
        ActionKwarg: "file_type",
      },
    },
  })

  // 4️⃣ Setup (cria BD, sincroniza, agenda scheduler)
  if err := h.Setup(ctx); err != nil {
    log.Fatal(err)
  }

  // 5️⃣ Servir HTTP com Gin
  log.Println("Iniciando servidor em :8080")
  router := h.GinRouter() // retorna *gin.Engine
  if err := router.Run(":8080"); err != nil {
    log.Fatal(err)
  }
}

// Handler da ação
func executeWithoutKafkaHandler(ctx context.Context, responses map[string]interface{}) (*helper.ActionResult, error) {
  fileType := responses["q1"].(string)

  if fileType == "CSV" {
    return &helper.ActionResult{
      Status:  "success",
      Message: "Arquivo CSV processado com sucesso",
      Data: map[string]interface{}{
        "file_id": "123",
        "url":     "s3://bucket/file-123.csv",
      },
    }, nil
  }

  return &helper.ActionResult{
    Status:  "error",
    Message: "Tipo de arquivo não suportado",
  }, nil
}
```

**Total de linhas:** ~60  
**Tempo de integração:** ~15 minutos  
**APIs disponíveis após:** ✅

- `GET /api/helper/v1/healthcare/`
- `GET /api/helper/v1/actions/`
- `GET /api/helper/v1/actions/execute-without-kafka`
- `POST /api/helper/v1/actions/execute-without-kafka/execute`

---

## Exemplo 2: Com Formulário Condicional

```go
package main

import (
  "context"
  "log"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  cfg := helper.NewConfig().
    WithDatabase("postgres://user:pass@localhost/vertdb").
    WithService("s3", &health_checks.S3Checker{Bucket: "my-bucket"})

  h, _ := helper.New(cfg)

  // ❗ Ação com formulário condicional
  h.RegisterAction("generate-document", &helper.Action{
    Name:        "Gerar Documento",
    Services:    []string{"S3"},
    Handler:     generateDocumentHandler,
    Questions: []*helper.Question{
      // Pergunta 1: Tipo de arquivo (raiz)
      {
        ID:          "q1",
        Label:       "O arquivo é CSV ou JSON?",
        Type:        "radio",
        Options:     []string{"CSV", "JSON"},
        IsRequired:  true,
        IsFirst:     true,
        ActionKwarg: "file_type",
      },
      // Pergunta 2a: Apareça apenas se q1 = "CSV"
      {
        ID:                "q2",
        Label:             "Você vai enviar arquivo ou URL?",
        Type:              "radio",
        Options:           []string{"Arquivo", "URL"},
        IsRequired:        true,
        ParentQuestionID:  stringPtr("q1"),
        ParentValue:       stringPtr("CSV"),
        ActionKwarg:       "csv_source",
      },
      // Pergunta 2b: Apareça apenas se q1 = "JSON"
      {
        ID:                "q3",
        Label:             "Qual o ID do workflow?",
        Type:              "text",
        IsRequired:        true,
        ParentQuestionID:  stringPtr("q1"),
        ParentValue:       stringPtr("JSON"),
        ActionKwarg:       "workflow_id",
      },
    },
  })

  h.Setup(context.Background())
  h.ServeHTTP(":8080")
}

func generateDocumentHandler(ctx context.Context, responses map[string]interface{}) (*helper.ActionResult, error) {
  // responses: {"q1": "CSV", "q2": "Arquivo", ...}
  fileType := responses["q1"].(string)
  csvSource := responses["q2"]

  if fileType == "CSV" {
    // Process CSV
    return &helper.ActionResult{
      Status:  "success",
      Message: "Documento gerado",
      Data: map[string]interface{}{
        "document_id": "doc-456",
      },
    }, nil
  }

  workflowID := responses["q3"].(string)
  // Process JSON
  return &helper.ActionResult{
    Status:  "success",
    Message: "Documento gerado para workflow " + workflowID,
    Data: map[string]interface{}{
      "workflow_id": workflowID,
    },
  }, nil
}

func stringPtr(s string) *string { return &s }
```

**Resultado na API:**

```json
GET /api/helper/v1/actions/generate-document

{
  "slug": "generate-document",
  "name": "Gerar Documento",
  "questions": [
    {
      "id": "q1",
      "label": "O arquivo é CSV ou JSON?",
      "type": "radio",
      "options": ["CSV", "JSON"],
      "is_required": true,
      "parent_question": null,
      "parent_value": null
    },
    {
      "id": "q2",
      "label": "Você vai enviar arquivo ou URL?",
      "type": "radio",
      "parent_question": "q1",
      "parent_value": "CSV"
    },
    {
      "id": "q3",
      "label": "Qual o ID do workflow?",
      "type": "text",
      "parent_question": "q1",
      "parent_value": "JSON"
    }
  ]
}
```

Frontend pode montar formulário condicional baseado em `parent_question` + `parent_value`.

---

## Exemplo 3: Health Checks Customizados

```go
package main

import (
  "context"
  "database/sql"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

// ✅ Qualquer tipo que implemente interface pode ser health check
type MyCustomChecker struct {
  ConnectionString string
}

func (c *MyCustomChecker) Check(ctx context.Context) (*helper.HealthCheckResult, error) {
  db, err := sql.Open("postgres", c.ConnectionString)
  if err != nil {
    return &helper.HealthCheckResult{
      Status:  "FAILED",
      Message: "Conexão falhou: " + err.Error(),
    }, nil
  }
  defer db.Close()

  if err := db.PingContext(ctx); err != nil {
    return &helper.HealthCheckResult{
      Status:  "FAILED",
      Message: "Ping falhou: " + err.Error(),
    }, nil
  }

  // Custom logic: checar versão, replicação, etc
  var version string
  if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
    return &helper.HealthCheckResult{
      Status:  "UNKNOWN",
      Message: "Não foi possível checar versão",
    }, nil
  }

  return &helper.HealthCheckResult{
    Status:  "OK",
    Message: "PostgreSQL " + version,
  }, nil
}

func main() {
  cfg := helper.NewConfig().
    WithDatabase("postgres://...").
    WithService("my-custom-postgres", &MyCustomChecker{
      ConnectionString: "postgres://user:pass@custom-host/db",
    })

  h, _ := helper.New(cfg)
  h.Setup(context.Background())
  h.ServeHTTP(":8080")
}
```

**Resultado:**

```json
GET /api/helper/v1/healthcare/

{
  "my-custom-postgres": {
    "status": "OK",
    "message": "PostgreSQL 14.5",
    "last_updated": "2026-07-20T14:30:00Z"
  }
}
```

---

## Exemplo 4: Integração com Projeto Gin Existente

```go
package main

import (
  "context"
  "log"

  "github.com/gin-gonic/gin"
  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  // Setup helper
  cfg := helper.NewConfig().
    WithDatabase("postgres://...").
    WithService("postgres", &health_checks.PostgresChecker{
      Host: "localhost",
    })

  h, _ := helper.New(cfg)
  h.Setup(context.Background())

  // ✅ Opção 1: Usar o router Gin do helper diretamente
  router := h.GinRouter()
  
  // Adicionar seus endpoints customizados
  router.GET("/my-app/users", listUsers)
  router.POST("/my-app/users", createUser)
  
  router.Run(":8080")
}

// Ou

// ✅ Opção 2: Mesclar com seu router Gin existente
func main2() {
  cfg := helper.NewConfig().
    WithDatabase("postgres://...")
  h, _ := helper.New(cfg)
  h.Setup(context.Background())

  // Seu router Gin
  r := gin.Default()
  
  // Integrar endpoints helper
  helperRouter := h.GinRouter()
  r.Group("/api/helper").Any("/*action", gin.WrapF(
    func(w http.ResponseWriter, req *http.Request) {
      helperRouter.ServeHTTP(w, req)
    }))
  
  // Seus endpoints
  r.GET("/my-app/users", listUsers)
  
  r.Run(":8080")
}

func listUsers(c *gin.Context) {
  // Seu código
}
```

**Benefício:** Helper library é agnóstica, funciona com qualquer framework HTTP.

---

## Exemplo 5: Testing

```go
package main

import (
  "context"
  "testing"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
)

func TestExecuteWithoutKafka(t *testing.T) {
  // Arrange
  result, err := executeWithoutKafkaHandler(context.Background(), map[string]interface{}{
    "q1": "CSV",
  })

  // Assert
  if err != nil {
    t.Fatal(err)
  }

  if result.Status != "success" {
    t.Errorf("esperado success, obteve %s", result.Status)
  }

  if result.Data["file_id"] != "123" {
    t.Errorf("file_id incorreto: %v", result.Data["file_id"])
  }
}

func TestHelperSetup(t *testing.T) {
  cfg := helper.NewConfig().
    WithDatabase("postgres://test:test@localhost/test_vertdb")

  h, err := helper.New(cfg)
  if err != nil {
    t.Fatal(err)
  }

  if err := h.Setup(context.Background()); err != nil {
    t.Fatal(err)
  }

  // Verificar que setup funcionou
  services, _ := h.ListServices(context.Background())
  if len(services) == 0 {
    t.Error("nenhum serviço foi criado")
  }
}
```

---

## Exemplo 6: Configuração com Variáveis de Ambiente

```go
package main

import (
  "os"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  cfg := helper.NewConfig().
    WithDatabase(os.Getenv("DATABASE_URL")).
    WithService("postgres", &health_checks.PostgresChecker{
      Host:     os.Getenv("POSTGRES_HOST"),
      Port:     5432,
      Database: os.Getenv("POSTGRES_DB"),
      User:     os.Getenv("POSTGRES_USER"),
      Password: os.Getenv("POSTGRES_PASSWORD"),
    }).
    WithService("s3", &health_checks.S3Checker{
      Bucket: os.Getenv("S3_BUCKET"),
      Region: os.Getenv("AWS_REGION"),
    })

  h, _ := helper.New(cfg)
  h.Setup(context.Background())
  h.ServeHTTP(":8080")
}
```

**env.example:**
```bash
DATABASE_URL=postgres://user:pass@localhost/vertdb
POSTGRES_HOST=localhost
POSTGRES_DB=vertdb
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
S3_BUCKET=my-bucket
AWS_REGION=us-east-1
```

---

## Exemplo 7: Com Scheduler Desativado

```go
package main

import (
  "context"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  cfg := helper.NewConfig().
    WithDatabase("postgres://...").
    WithService("postgres", &health_checks.PostgresChecker{Host: "localhost"}).
    WithSchedulerDisabled() // ❌ Não agenda health checks periódicos

  h, _ := helper.New(cfg)
  h.Setup(context.Background())

  // Neste caso:
  // ✅ Serviços são monitorados via API (GET /api/helper/v1/healthcare)
  // ❌ Sem execução automática periódica
  // ✅ Usuário pode chamar h.RunHealthChecks(ctx) manualmente se necessário

  h.ServeHTTP(":8080")
}
```

---

## Exemplo 8: Com Scheduler (Default)

```go
package main

import (
  "context"
  "time"

  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  cfg := helper.NewConfig().
    WithDatabase("postgres://...").
    WithService("postgres", &health_checks.PostgresChecker{Host: "localhost"}).
    WithHealthCheckInterval(10 * time.Minute). // Default: 10 minutos

  h, _ := helper.New(cfg)
  h.Setup(context.Background())

  // Neste caso:
  // ✅ Helper registra job com cron (a cada 10 minutos)
  // ✅ Health checks executam automaticamente
  // ✅ BD é atualizado com novos registros
  // ✅ APIs retornam dados mais recentes

  h.ServeHTTP(":8080")
  // Cron roda em background
}
```

---

## Exemplo 9: Health Check com Força de Refresh

```bash
# Sem refresh (retorna último resultado em BD)
$ curl http://localhost:8080/api/helper/v1/healthcare

{
  "postgres": {
    "status": "OK",
    "last_updated": "2026-07-20T14:25:00Z"
  }
}

# Com força refresh (executa check agora)
$ curl "http://localhost:8080/api/helper/v1/healthcare?force_refresh=true"

{
  "postgres": {
    "status": "OK",
    "last_updated": "2026-07-20T14:30:15Z"  # <-- atualizado
  }
}
```

---

## Resumo: O Que Quer Dizer "Plugin"?

A biblioteca é um **plugin** porque:

1. **Mínimas dependências**
   - Usuário adiciona: `import "github.com/caiofariavert/golang_vert_helper/pkg/helper"`
   - Pronto!

2. **Não força estrutura**
   - Funciona com net/http, Gin, Echo, qualquer framework
   - Usuário integra endpoints onde quiser

3. **Fácil customizar**
   - Health checks: implementa `HealthChecker` interface
   - Actions: escreve função, registra via `h.RegisterAction()`
   - Queries custom: estende service com métodos

4. **Isolado**
   - Toda lógica em pacotes `pkg/` e `internal/`
   - Não pisa em código do usuário
   - Usuário controla ciclo de vida (Setup, ServeHTTP)

5. **Type-safe**
   - Erros em compile time, não runtime
   - IDE fornece autocomplete
   - Refactor seguro

---

## Validação de Decisões

Com esses exemplos, podemos validar:

✅ Builder pattern é intuitivo e type-safe  
✅ Registry pattern é explícito sem ser verboso  
✅ Interface HealthChecker é fácil de implementar  
✅ ActionHandler signature é simples  
✅ Formulários condicionais funcionam bem  
✅ Integra com frameworks web existentes  
✅ Setup é idiomático Go  

