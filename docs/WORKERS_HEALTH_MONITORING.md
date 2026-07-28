# Workers Health Monitoring - Vert Helper Go

## 📋 Visão Geral

Este documento descreve como monitorar a saúde de workers, jobs, goroutines e outras tarefas assincronas rodando na aplicação usando a biblioteca Vert Helper.

**Caso de uso:** Monitorar se workers de fila (Kafka, RabbitMQ), processadores em background, consumers, etc estão funcionando corretamente.

---

## 🎯 Arquitetura

Workers podem ser monitorados através do padrão `HealthChecker` da biblioteca, que aceita qualquer tipo que implemente a interface.

```
Aplicação
├── Worker 1 (Kafka consumer)
│   └── State: "running", "paused", "failed"
├── Worker 2 (Background job processor)
│   └── State: "processing", "idle", "error"
├── Worker 3 (Webhook dispatcher)
│   └── State: "queued", "sending", "backoff"
└── Vert Helper Registry
    └── Monitora todos via health checks periódicos
```

---

## 💡 Estratégias de Implementação

### Estratégia 1: Registro de Callback (Simples)

Cada worker registra um callback que retorna seu estado.

```go
type WorkerRegistry struct {
  workers map[string]func(context.Context) string
}

func (r *WorkerRegistry) RegisterWorker(name string, statusFunc func(context.Context) string) {
  r.workers[name] = statusFunc
}

func (r *WorkerRegistry) CheckWorkers(ctx context.Context) (*helper.HealthCheckResult, error) {
  var failedWorkers []string
  
  for name, statusFunc := range r.workers {
    status := statusFunc(ctx)
    if status != "running" && status != "idle" {
      failedWorkers = append(failedWorkers, name)
    }
  }
  
  if len(failedWorkers) > 0 {
    return &helper.HealthCheckResult{
      Status:  "FAILED",
      Message: fmt.Sprintf("Workers failed: %s", strings.Join(failedWorkers, ", ")),
    }, nil
  }
  
  return &helper.HealthCheckResult{
    Status:  "OK",
    Message: fmt.Sprintf("All %d workers healthy", len(r.workers)),
  }, nil
}

// Implementa HealthChecker
func (r *WorkerRegistry) Check(ctx context.Context) (*helper.HealthCheckResult, error) {
  return r.CheckWorkers(ctx)
}
```

**Uso:**
```go
workerReg := &WorkerRegistry{workers: make(map[string]func(context.Context) string)}

// Registrar workers
workerReg.RegisterWorker("kafka-consumer", kafkaConsumer.Status)
workerReg.RegisterWorker("job-processor", jobProcessor.Status)

// Registrar como health check
cfg := helper.NewConfig().
  WithService("workers", workerReg)

h, _ := helper.New(cfg)
h.Setup(ctx)
```

---

### Estratégia 2: Estrutura Compartilhada (Recomendado)

Workers compartilham uma estrutura comum que registra seu estado.

```go
// domain/worker.go
type WorkerStatus string

const (
  WorkerRunning   WorkerStatus = "running"
  WorkerPaused    WorkerStatus = "paused"
  WorkerFailed    WorkerStatus = "failed"
  WorkerIdle      WorkerStatus = "idle"
  WorkerProcessing WorkerStatus = "processing"
  WorkerBackoff   WorkerStatus = "backoff"
)

type Worker struct {
  ID             string
  Name           string
  Status         WorkerStatus
  LastCheck      time.Time
  LastError      *string
  ProcessedCount int64
  FailedCount    int64
}

type WorkerPool struct {
  workers sync.Map // thread-safe map[string]*Worker
}

func (p *WorkerPool) Register(worker *Worker) {
  p.workers.Store(worker.ID, worker)
}

func (p *WorkerPool) UpdateStatus(id string, status WorkerStatus, err error) {
  if w, ok := p.workers.Load(id); ok {
    worker := w.(*Worker)
    worker.Status = status
    worker.LastCheck = time.Now()
    if err != nil {
      errMsg := err.Error()
      worker.LastError = &errMsg
    }
  }
}

func (p *WorkerPool) GetAll() []*Worker {
  var workers []*Worker
  p.workers.Range(func(key, value interface{}) bool {
    workers = append(workers, value.(*Worker))
    return true
  })
  return workers
}

// HealthChecker implementation
func (p *WorkerPool) Check(ctx context.Context) (*helper.HealthCheckResult, error) {
  workers := p.GetAll()
  
  if len(workers) == 0 {
    return &helper.HealthCheckResult{
      Status:  "UNKNOWN",
      Message: "No workers registered",
    }, nil
  }
  
  var failed []string
  for _, w := range workers {
    if w.Status == WorkerFailed {
      failed = append(failed, fmt.Sprintf("%s (error: %s)", w.Name, *w.LastError))
    }
  }
  
  if len(failed) > 0 {
    return &helper.HealthCheckResult{
      Status:  "FAILED",
      Message: fmt.Sprintf("Failed workers: %s", strings.Join(failed, "; ")),
    }, nil
  }
  
  return &helper.HealthCheckResult{
    Status:  "OK",
    Message: fmt.Sprintf("%d workers running", len(workers)),
  }, nil
}
```

**Uso em Worker:**
```go
type KafkaConsumer struct {
  workerPool *WorkerPool
  id         string
  // ... outros campos
}

func (kc *KafkaConsumer) Run(ctx context.Context) {
  worker := &Worker{
    ID:     kc.id,
    Name:   "Kafka Consumer",
    Status: WorkerRunning,
  }
  kc.workerPool.Register(worker)
  
  for {
    select {
    case <-ctx.Done():
      kc.workerPool.UpdateStatus(kc.id, WorkerPaused, nil)
      return
    default:
      msg, err := kc.consumer.ReadMessage(ctx)
      if err != nil {
        kc.workerPool.UpdateStatus(kc.id, WorkerFailed, err)
        // retry logic
        continue
      }
      
      kc.workerPool.UpdateStatus(kc.id, WorkerProcessing, nil)
      if err := kc.processMessage(msg); err != nil {
        kc.workerPool.UpdateStatus(kc.id, WorkerFailed, err)
      } else {
        kc.workerPool.UpdateStatus(kc.id, WorkerRunning, nil)
      }
    }
  }
}
```

---

### Estratégia 3: API Endpoint Dedicado (Avançado)

Expor um endpoint `/api/helper/v1/workers` que retorna detalhes dos workers.

```go
// internal/adapters/http/handlers.go
type WorkerDetail struct {
  ID             string    `json:"id"`
  Name           string    `json:"name"`
  Status         string    `json:"status"`
  LastCheck      time.Time `json:"last_check"`
  LastError      *string   `json:"last_error,omitempty"`
  ProcessedCount int64     `json:"processed_count"`
  FailedCount    int64     `json:"failed_count"`
  Uptime         int64     `json:"uptime_seconds"`
}

func (h *Handler) GetWorkers(c *gin.Context) {
  ctx := c.Request.Context()
  
  workers := h.workerPool.GetAll()
  details := make([]WorkerDetail, len(workers))
  
  for i, w := range workers {
    details[i] = WorkerDetail{
      ID:             w.ID,
      Name:           w.Name,
      Status:         string(w.Status),
      LastCheck:      w.LastCheck,
      LastError:      w.LastError,
      ProcessedCount: w.ProcessedCount,
      FailedCount:    w.FailedCount,
      Uptime:         int64(time.Since(w.StartTime).Seconds()),
    }
  }
  
  c.JSON(http.StatusOK, details)
}

func (h *Handler) GetWorkerDetail(c *gin.Context) {
  id := c.Param("id")
  ctx := c.Request.Context()
  
  workers := h.workerPool.GetAll()
  for _, w := range workers {
    if w.ID == id {
      c.JSON(http.StatusOK, WorkerDetail{
        ID:             w.ID,
        Name:           w.Name,
        Status:         string(w.Status),
        LastCheck:      w.LastCheck,
        LastError:      w.LastError,
        ProcessedCount: w.ProcessedCount,
        FailedCount:    w.FailedCount,
      })
      return
    }
  }
  
  c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
}

// Registrar routes no Gin
func (h *Handler) RegisterWorkerRoutes(router *gin.Engine) {
  workers := router.Group("/api/helper/v1/workers")
  {
    workers.GET("", h.GetWorkers)
    workers.GET("/:id", h.GetWorkerDetail)
  }
}
```

**API Response:**
```json
GET /api/helper/v1/workers

[
  {
    "id": "kafka-consumer-1",
    "name": "Kafka Consumer",
    "status": "running",
    "last_check": "2026-07-20T14:30:15Z",
    "processed_count": 15234,
    "failed_count": 0,
    "uptime_seconds": 3600
  },
  {
    "id": "job-processor-1",
    "name": "Job Processor",
    "status": "failed",
    "last_check": "2026-07-20T14:30:10Z",
    "last_error": "connection timeout",
    "processed_count": 542,
    "failed_count": 3,
    "uptime_seconds": 7200
  }
]
```

---

## 🔄 Alertas e Ações Automáticas

Quando um worker falha, a biblioteca pode:

### 1. Registrar em BD (ActionExecution)
```go
// Execução automática de ação em falha
func (h *HealthService) onWorkerFailure(worker *Worker, err error) {
  // Opção: executar ação de recuperação automática
  h.actionService.ExecuteAction(ctx, "recover-worker", map[string]interface{}{
    "worker_id": worker.ID,
    "error":     err.Error(),
  })
}
```

### 2. Webhook/Callback
```go
type HealthCheckHook func(serviceHealth *ServiceHealth) error

// Configurar hook de falha
h.OnHealthCheckFailure("workers", func(sh *ServiceHealth) error {
  // Enviar email, webhook, etc
  return notifyOps(sh)
})
```

### 3. Ação Recomendada na API
```json
GET /api/helper/v1/healthcare

{
  "workers": {
    "status": "FAILED",
    "message": "2 workers failed",
    "last_updated": "2026-07-20T14:30:15Z",
    "is_recommended": true
  }
}

GET /api/helper/v1/actions?service=workers

[
  {
    "slug": "restart-failed-workers",
    "name": "Reiniciar Workers Falhados",
    "is_recommended": true
  },
  {
    "slug": "view-worker-logs",
    "name": "Visualizar Logs",
    "is_recommended": true
  }
]
```

---

## 📊 Armazenamento em BD

Opcionalmente, armazenar histórico de workers em BD.

```sql
CREATE TABLE worker_snapshots (
  id UUID PRIMARY KEY,
  service_id UUID NOT NULL REFERENCES services(id),
  worker_id VARCHAR(255),
  worker_name VARCHAR(255),
  status VARCHAR(50),
  error_message TEXT,
  processed_count INT,
  failed_count INT,
  uptime_seconds INT,
  taken_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_worker_snapshots_service_taken_at 
  ON worker_snapshots(service_id, taken_at DESC);
```

```go
// Persistir snapshot de workers periodicamente
func (s *HealthService) SnapshotWorkers(ctx context.Context) error {
  workers := s.workerPool.GetAll()
  
  for _, w := range workers {
    snapshot := &WorkerSnapshot{
      ID:               uuid.New(),
      ServiceID:        serviceID,
      WorkerID:         w.ID,
      WorkerName:       w.Name,
      Status:           string(w.Status),
      ErrorMessage:     w.LastError,
      ProcessedCount:   w.ProcessedCount,
      FailedCount:      w.FailedCount,
      UptimeSeconds:    int(time.Since(w.StartTime).Seconds()),
      TakenAt:          time.Now(),
    }
    
    if err := s.repo.CreateWorkerSnapshot(ctx, snapshot); err != nil {
      s.logger.Error("failed to snapshot worker", "error", err)
    }
  }
  
  return nil
}
```

---

## 🛠️ Exemplo Completo: Kafka + Job Processor

```go
package main

import (
  "context"
  "log"
  "time"
  
  "github.com/caiofariavert/golang_vert_helper/pkg/helper"
  "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

func main() {
  // Setup helper
  cfg := helper.NewConfig().
    WithDatabase("postgres://...").
    WithService("postgres", &health_checks.PostgresChecker{...})
  
  h, _ := helper.New(cfg)
  
  // Setup worker pool
  workerPool := &WorkerPool{workers: sync.Map{}}
  
  // Registrar workers como health check
  h.RegisterService("workers", workerPool)
  
  h.Setup(context.Background())
  
  // Iniciar workers em goroutines
  kafkaConsumer := NewKafkaConsumer(workerPool)
  go kafkaConsumer.Run(context.Background())
  
  jobProcessor := NewJobProcessor(workerPool)
  go jobProcessor.Run(context.Background())
  
  // Scheduler monitora workers a cada 10 segundos
  h.WithHealthCheckInterval(10 * time.Second)
  
  // Servir APIs
  router := h.GinRouter()
  router.GET("/api/helper/v1/workers", func(c *gin.Context) {
    // Handler de workers
  })
  router.Run(":8080")
}
```

---

## 📈 Monitoramento em Dashboard

Frontend pode consultar periodicamente:

```javascript
// Frontend (React/Vue/Angular)
async function refreshWorkerStatus() {
  // Saúde geral
  const health = await fetch('/api/helper/v1/healthcare');
  const healthData = await health.json();
  
  // Detalhes de workers
  const workers = await fetch('/api/helper/v1/workers');
  const workerData = await workers.json();
  
  // Mostrar status visual (verde/amarelo/vermelho)
  updateDashboard({
    workersHealth: healthData.workers.status,
    workerDetails: workerData,
  });
}

// Atualizar a cada 30 segundos
setInterval(refreshWorkerStatus, 30000);
```

---

## ✅ Checklist de Implementação

- [ ] Definir estrutura `WorkerStatus` e `Worker`
- [ ] Implementar `WorkerPool` com sync.Map
- [ ] Implementar `HealthChecker` para workers
- [ ] Handlers HTTP para `/api/helper/v1/workers`
- [ ] Opcional: Armazenar snapshots em BD
- [ ] Opcional: Webhook/email em falha
- [ ] Testes de stress com múltiplos workers
- [ ] Documentação para usuários

---

## 🚀 Próximos Passos (V2)

- [ ] Dashboard web para visualizar workers
- [ ] Histórico de transições de estado
- [ ] Alertas configuráveis por worker
- [ ] Métricas Prometheus
- [ ] Distributed tracing
- [ ] Workers em múltiplos servidores (discovery)
