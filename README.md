# Vert Helper Go

Biblioteca Go para monitoramento de saúde de serviços e execução de ações interativas.

📖 **[Ver Manual de Uso completo](docs/manual_de_uso.md)**

---

## Instalação rápida

```bash
go get github.com/caiofariavert/golang_vert_helper
```

## Exemplo mínimo

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/caiofariavert/golang_vert_helper/pkg/helper"
    healthchecks "github.com/caiofariavert/golang_vert_helper/pkg/health_checks"
)

h := helper.New(db)
h.RegisterService("postgres", healthchecks.NewGormPostgresChecker(db))

router := gin.Default()
h.RegisterRoutes(router, db, nil)
router.Run(":8080")
```

## Autenticacao JWT (Bearer)

As rotas em `/api/helper/v1/healthcare`, `/api/helper/v1/actions` e `/api/helper/v1/workers` agora exigem token JWT no header:

```text
Authorization: Bearer <token>
```

Rota publica para autenticacao:

```text
POST /api/helper/v1/auth/
```

Payload:

```json
{
    "email": "helper@vert-capital.com",
    "password": "Helper@123"
}
```

Usuario padrao criado automaticamente no startup (ou atualizado) a partir de variaveis de ambiente:

```bash
HELPER_API_AUTH_EMAIL=helper@vert-capital.com
HELPER_API_AUTH_PASSWORD=Helper@123
```

Opcionalmente, defina segredo e TTL do token:

```bash
HELPER_JWT_SECRET=troque-este-segredo
HELPER_JWT_TTL_MINUTES=60
```

Para documentação completa, exemplos avançados e referência da API, consulte o **[Manual de Uso](docs/manual_de_uso.md)**.

## Importante para integração externa

Ao usar esta biblioteca em outro projeto, **nao importe pacotes `internal/*`**.
Use os tipos publicos em `pkg/contracts` quando precisar de structs, enums e definicoes de sync.

Exemplo:

```go
import "github.com/caiofariavert/golang_vert_helper/pkg/contracts"
```
