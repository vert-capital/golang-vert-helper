# Decisões Técnicas - Vert Helper Go

## Resumo Executivo

| Aspecto | Decisão | Justificativa | Risco |
|---------|---------|---------------|-------|
| **Framework HTTP** | `gin` ✅ | Performance, API simples, ecosystem maduro | 1 dependência |
| **Database** | PostgreSQL + GORM ✅ | ORM maduro, múltiplos BDs, hooks, plugin-ready | 2 dependências vs 1 (lib/pq) |
| **Scheduler** | `robfig/cron/v3` ✅ | Mencionado na spec, maduro, familiar | 1 dependência |
| **Logging** | `slog` (stdlib 1.21+) ✅ | Estruturado nativo, sem dependências | Requer Go 1.21+ |
| **Validação** | `validator/v10` ✅ | Maduro, bem mantido, simples | 1 dependência |
| **Migrations** | `golang-migrate` ✅ | Padrão ouro em Go | 1 dependência |
| **Config** | Builder Pattern ✅ | Type-safe, idiomático Go, validado | Mais verbose que YAML |
| **Actions** | Registry Pattern ✅ | Explícito, sem magic, type-safe, validado | Não usa reflection |
| **Health Checks** | Interface `HealthChecker` ✅ | Fácil de estender, plugin-ready | Permite qualquer implementação |
| **Multi-DB Support** | GORM permite múltiplos BDs ✅ | Usuários estendem com suas próprias conexões | Mantém PostgreSQL como padrão |

---

## Filosofia de Design

### 1. **Simplicidade para o Usuário Final**

```go
// ✅ Objetivo: integrar em ~30 linhas
cfg := helper.NewConfig().
  WithDatabase("postgres://...").
  WithService("postgres", &PostgresChecker{...}).
  WithService("s3", &S3Checker{...})

h, _ := helper.New(cfg)

h.RegisterAction("my-action", &helper.Action{
  Handler: myFunc,
  Questions: [...],
})

h.Setup(ctx)

router := h.GinRouter() // *gin.Engine
router.Run(":8080")
```

### 2. **Zero Magic, Máximo Explicitness**

- ❌ Não usar reflection para descobrir ações (como decorator Python)
- ✅ Registry pattern explícito
- ✅ Type-safe em compile time
- ✅ Erros claros em tempo de build

### 3. **Plugin-Ready**

- ✅ Gin handlers (integra com qualquer projeto Gin)
- ✅ Mínimas dependências (8 principais)
- ✅ Isolado (não força estrutura do projeto)
- ✅ GORM permite múltiplos BDs (PostgreSQL, MySQL, SQLite, etc)

### 4. **Multi-Database Support (Pronto para Expansão)**

**V1:** PostgreSQL como padrão (recomendado)  
**Suporte:** GORM permite usuários adicionar suas próprias conexões de BD

```go
// Usuário pode registrar múltiplos BDs
cfg := helper.NewConfig().
  WithDatabase("postgres://main").
  WithCustomDB("mysql", &MySQLConnection{...}).
  WithCustomDB("sqlite", &SQLiteConnection{...})
```

---

## Trade-offs e Compromissos

### Builder Pattern vs YAML Config

| Builder (Escolhido) | YAML |
|-----------|------|
| ✅ Type-safe em build | ❌ Runtime type-unsafe |
| ✅ IDE autocomplete | ✅ Menos código |
| ✅ Imutável | ✅ Familiar a DevOps |
| ❌ Mais verbose | ❌ Requer parser |

**Decisão:** Builder como padrão + suporte opcional a YAML (usando `viper`) futuramente

---

### Registry vs Reflection/Scanning

| Registry (Escolhido) | Reflection/Scanning |
|---------|-------------|
| ✅ Explícito, debugável | ❌ Magic, hard to trace |
| ✅ Type-safe | ❌ Runtime errors |
| ❌ Mais verbose | ✅ Menos código |
| ✅ Fast startup | ❌ Reflection overhead |

**Decisão:** Registry explícito (como Go gosta)

**Alternativa futura:** Gerador de código que lê tags e gera registry

---

### Gin vs Net/HTTP Puro

| Gin (Escolhido) ✅ | Net/HTTP Puro |
|-----------|---------|
| ✅ Performance excelente | ✅ Zero dependências |
| ✅ API intuitiva | ✅ Simples para casos triviais |
| ✅ Middleware ecosystem | ❌ Mais boilerplate |
| ✅ Fácil integração | ❌ Comunidade menor |
| ✅ Comunidade forte | ✅ Go idiomático |

**Decisão:** Gin (validado pelo usuário)

**Justificativa:** Framework maduro, performance, comunidade forte, fácil para integrar em projetos existentes

**Bridge:** Helper expõe handlers prontos para Gin

```go
// Usar diretamente com Gin
router := h.GinRouter() // retorna *gin.Engine
router.Run(":8080")
e.GET("/api/helper/v1/healthcare", helper.HealthcareHandler())

// Ou net/http puro
mux.HandleFunc("/api/helper/v1/healthcare", helper.HealthcareHandler())
```

---

### GORM vs Sqlc vs Raw SQL

| GORM (Escolhido) ✅ | Sqlc | Raw SQL |
|--------|------|---------|
| ✅ ORM maduro | ✅ Type-safe compile | ✅ Total controle |
| ✅ Múltiplos BDs | ✅ Zero overhead | ❌ SQL injection risk |
| ✅ Hooks para lógica | ❌ Curva aprendizado | ✅ Simples |
| ✅ Plugin-ready | ✅ Fast queries | ✅ Fast |
| ❌ Pequeno overhead | - | - |

**Decisão:** GORM (validado pelo usuário)

**Justificativa:** Maduro, suporta múltiplos BDs, permite usuários estenderem com suas conexões, hooks para business logic

**Fluxo:** GORM models → Migrations → Database operations

---

## Estrutura de Pacotes Explicada

```
pkg/helper/
├── helper.go         // API principal
├── config.go         // Builder de config
├── registry.go       // Registry de ações
└── types.go          // Tipos públicos

internal/
├── domain/           // Entidades + contratos
├── adapters/         // PostgreSQL, HTTP, Cron
├── services/         // Lógica de negócio
└── repository/       // Data access
```

**Segregação:**
- `pkg/` = público, versionado, backward-compatible
- `internal/` = privado, pode mudar entre versões

---

## Padrões Go Utilizados

### 1. **Repository Pattern**
```go
type ServiceRepository interface {
  Create(ctx context.Context, s *Service) error
  GetByName(ctx context.Context, name string) (*Service, error)
}
```

**Por quê:** Desacopla domínio de implementação

---

### 2. **Dependency Injection**
```go
type HealthService struct {
  repo ServiceRepository
  logger *slog.Logger
}

func NewHealthService(repo ServiceRepository, logger *slog.Logger) *HealthService {
  return &HealthService{repo, logger}
}
```

**Por quê:** Testável, flexível, inverso de controle

---

### 3. **Interface Segregation**
```go
type HealthChecker interface {
  Check(ctx context.Context) (*HealthCheckResult, error)
}
```

**Por quê:** Usuário implementa interface mínima

---

### 4. **Error Handling Explícito**
```go
if err := service.Execute(ctx); err != nil {
  if errors.Is(err, ErrActionNotFound) {
    // Handle específico
  }
}
```

**Por quê:** Go standard, sem exceptions/try-catch

---

### 5. **Context Propagation**
```go
func (s *Service) Execute(ctx context.Context) error {
  // Timeout, cancellation, values fluem naturalmente
}
```

**Por quê:** Controle de ciclo de vida, goroutines, deadlines

---

## Checklist de Implementação

### Estrutura
- [ ] Criar `go.mod` + definir módulo
- [ ] Criar estrutura de pastas (pkg/, internal/, cmd/)
- [ ] Setup .gitignore

### Domain
- [ ] Definir entities (Service, Action, etc)
- [ ] Definir interfaces/contracts (HealthChecker, ActionHandler)
- [ ] Definir types resposta (ActionResult, HealthCheckResult)

### Database
- [ ] Migrations SQL (sqlc)
- [ ] Queries SQL files (.sql)
- [ ] Generate via sqlc
- [ ] Implementar repositories

### Services
- [ ] HealthService (orquestrar checks)
- [ ] ActionService (orquestrar execução)
- [ ] SyncService (sincronizar BD)

### Adapters
- [ ] PostgreSQL connection pool
- [ ] HTTP handlers (mux puro)
- [ ] Cron scheduler
- [ ] Logging (slog)

### Public API
- [ ] Config builder
- [ ] Registry de ações
- [ ] Main helper struct
- [ ] Setup function

### Exemplos
- [ ] Exemplo mínimo (10 linhas)
- [ ] Exemplo com scheduler
- [ ] Exemplo com health checks custom

### Testes
- [ ] Unit tests (repositories, services)
- [ ] Integration tests (database)
- [ ] Handler tests (HTTP)

### Documentação
- [ ] README.md
- [ ] ARQUITETURA.md (este)
- [ ] Guia de integração
- [ ] Exemplos comentados

---

## Próximas Discussões

1. **Suportar múltiplos bancos?** (v2)
   - Sim: implementar interface Repository
   - Não: apenas PostgreSQL v1

2. **YAML config ou só builder?**
   - Builder: padrão, type-safe
   - YAML: add depois se necessário

3. **Autenticação/permissões v1 ou v2?**
   - V1: AllowAll ou interface PermissionChecker simples
   - V2: JWT, OAuth2, etc

4. **Webhooks em falha de saúde?**
   - V1: não
   - V2: sim (callback pattern)

5. **UI web para gerenciar ações?**
   - V1: não
   - V2: sim (painel administrativo)

---

## Status

- **Documento criado:** 2026-07-20
- **Fase:** Planejamento ✅
- **Próximo:** Validação e início implementação
