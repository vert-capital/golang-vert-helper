# Especificação Técnica Agnóstica de Framework

## 1. Objetivo

Este documento define um blueprint técnico para uma biblioteca de:

- Monitoramento de saúde de serviços externos
- Catálogo e execução de ações operacionais
- Formulários condicionais para entrada de dados
- Exposição de APIs para consumo por frontends e automações
- Agendamento de rotinas periódicas

A especificação é independente de framework e linguagem, permitindo implementação em Python, Node.js, Java, Go, etc.

---

## 2. Escopo Funcional

### 2.1 Health Monitoring

A biblioteca deve:

- Ler serviços monitorados de configuração
- Executar verificações de saúde por serviço
- Persistir histórico de verificações
- Retornar o estado mais recente por serviço
- Permitir execução sob demanda via API
- Permitir execução periódica via scheduler

### 2.2 Action Catalog

A biblioteca deve:

- Registrar ações "codificadas" na aplicação
- Sincronizar catálogo persistido com ações registradas em código
- Suportar associação de ações a serviços (recomendação por falha)
- Executar ações via API
- Registrar histórico/auditoria de execução

### 2.3 Conditional Forms

A biblioteca deve:

- Associar perguntas a uma ação
- Permitir fluxo condicional por relação pai/valor
- Mapear respostas para argumentos da função de ação

### 2.4 Operational Security Scan (CI)

A esteira deve:

- Executar varredura de filesystem
- Executar varredura de imagem construída
- Falhar pipeline para severidade HIGH/CRITICAL

---

## 3. Arquitetura de Referência

## 3.1 Camadas

### Core Domain

- Entidades e regras de negócio
- Contratos para health checks, actions e formulários

### Application Services

- Casos de uso:
  - Sincronização de serviços
  - Sincronização de ações
  - Execução de health checks
  - Limpeza de histórico
  - Execução de ação

### Infrastructure Adapters

- Banco de dados
- Scheduler
- Health check adapters (Postgres/Kafka/S3)
- API transport (HTTP)

### Interface/API

- Endpoints REST
- Serialização e validação de payload

---

## 4. Modelo de Dados Canônico

### 4.1 Service

Campos mínimos:

- id (UUID)
- name (string única)
- is_active (boolean para soft delete)
- created_at
- updated_at

Regras:

- Serviços removidos da configuração devem virar is_active=false
- Histórico deve ser preservado

### 4.2 ServiceHealth

Campos mínimos:

- id (UUID)
- service_id (FK)
- status (OK, FAILED, UNKNOWN)
- message (nullable)
- checked_at
- created_at

Regras:

- Manter histórico completo
- API default deve retornar apenas o registro mais recente por serviço

### 4.3 Action

Campos mínimos:

- id (UUID)
- slug (único)
- name
- description
- function_path ou handler_id
- status (active/inactive)
- metadata (JSON opcional)
- created_at
- updated_at

Relacionamentos:

- Action <-> Service (N:N)

### 4.4 Question

Campos mínimos:

- id (UUID)
- action_id (FK)
- label
- type (radio/text/textarea/file/select)
- options (JSON opcional)
- is_required
- parent_question_id (nullable)
- parent_value (nullable)
- action_kwarg (nullable)
- is_first
- created_at
- updated_at

Regras:

- Fluxo começa em perguntas is_first=true
- Próxima pergunta é selecionada por parent_question_id + parent_value
- Resposta pode ser mapeada para argumento de execução via action_kwarg

### 4.5 ActionExecution

Campos mínimos:

- id (UUID)
- action_id (FK)
- responses (JSON)
- result (JSON)
- executed_by (nullable)
- executed_at
- created_at

Regras:

- Toda execução deve ser auditável
- resultado deve ter status e message

---

## 5. Contratos Técnicos

### 5.1 Contrato de Health Check

Entrada:

- context (objeto/dicionário)

Saída aceita:

- objeto { status, message? }
- tupla (status, message)
- string status

Status válidos:

- OK
- FAILED
- UNKNOWN

Comportamento padrão:

- retorno inválido => UNKNOWN
- exceção => FAILED + mensagem

### 5.2 Contrato de Action Handler

Entrada:

- responses: mapa pergunta_id -> resposta

Saída:

- status: success|error|info
- message: string
- data: objeto opcional
- steps: lista opcional

### 5.3 Contrato de Registro de Action em Código

Cada action registrada em código deve conter, no mínimo:

- slug
- name
- description
- services (lista de nomes de serviço)
- function_path (ou identificador equivalente do handler)
- function (referência executável)

Opcionalmente, pode conter `questions`, no formato:

```
[
  {
    "label": "Pergunta raiz",
    "type": "radio",
    "options": ["A", "B"],
    "is_required": true,
    "action_kwarg": "arg_name",
    "children": [
      {
        "label": "Pergunta filha",
        "type": "text",
        "is_required": true,
        "action_kwarg": "child_arg",
        "parent_value": "A"
      }
    ]
  }
]
```

Regras:

- `children` define relacionamento recursivo pai/filho
- `parent_value` na filha define quando ela deve ser ativada
- `action_kwarg` define o nome do argumento de entrada do handler

### 5.4 Contrato de Scheduler Adapter

Interface mínima:

- register(interval_seconds): string

Responsabilidades:

- Registrar job de health check periódico
- Registrar job de limpeza de logs (exemplo: 24h)
- Atualizar jobs existentes de forma idempotente

---

## 6. API Contract (Agnóstico)

### 6.1 GET /api/helper/v1/healthcare

Comportamento:

- Retorna estado mais recente por serviço ativo
- Query opcional force_refresh=true para executar checks em tempo real

Resposta exemplo:

{
  "S3": {
    "status": "OK",
    "last_updated": "2026-07-16T12:00:00Z"
  },
  "POSTGRES": {
    "status": "FAILED",
    "message": "timeout",
    "last_updated": "2026-07-16T12:00:01Z"
  }
}

### 6.2 GET /api/helper/v1/actions

Comportamento:

- Paginação
- Filtro por service
- Search por nome
- Ordenação com recomendadas primeiro

Recomendação:

- is_recommended=true quando algum serviço associado está FAILED

### 6.3 GET /api/helper/v1/actions/{slug}

Comportamento:

- Retorna metadados da ação
- Retorna todas as perguntas do formulário condicional

### 6.4 POST /api/helper/v1/actions/{slug}/execute

Payload exemplo:

{
  "questions": {
    "q1": "CSV",
    "q2": "URL"
  }
}

Resposta exemplo:

{
  "status": "success",
  "message": "Ação executada",
  "data": {}
}

### 6.5 GET /api/helper/v1/app-health

Objetivo:

- Endpoint de saúde da aplicação sem depender do runtime do framework principal

Estratégia recomendada:

- Arquivo JSON estático atualizado por job externo

Requisito de exposição HTTP:

- A exposição pública de /api/helper/v1/app-health deve ser feita por Nginx (ou gateway compatível com semântica equivalente de arquivo estático)
- O Nginx deve responder diretamente o arquivo estático (ex: /app/health.json), sem encaminhar ao runtime da aplicação
- O endpoint deve permanecer disponível mesmo com indisponibilidade parcial do framework principal

Recomendação de execução em container:

- O processo principal da aplicação deve executar como usuário não-root
- A atualização periódica do JSON pode usar:
  - scheduler externo (Kubernetes CronJob, sidecar, host scheduler), ou
  - loop em background no entrypoint/comando do container (ex: a cada 600s)
- Evitar depender de cron de sistema no runtime quando isso exigir root

Exemplo de location Nginx:

location /api/helper/v1/app-health/ {
    alias /app/health.json;
    default_type application/json;
    add_header Cache-Control "no-store";
}

---

## 7. Configuração Canônica

Campos recomendados:

- PERMISSION_CLASS
- SERVICES
- SCHEDULER
- HEALTH_CHECK_INTERVAL
- HEALTH_LOG_RETENTION_DAYS

Exemplo lógico:

- SERVICES.<service_name>.function
- SERVICES.<service_name>.context

---

## 8. Sincronização e Idempotência

## 8.1 Setup Command (orquestrador)

Deve executar:

- sync_services
- sync_actions
- register_scheduler

## 8.2 Regras de Sync de Services

- Cria serviços novos de configuração
- Reativa serviços existentes
- Soft delete serviços removidos

## 8.3 Regras de Sync de Actions

- Descobre handlers registrados em código
- Cria/atualiza catálogo persistido
- Remove órfãs (não existem mais no código)
- Sincroniza perguntas declaradas em `questions` com estrutura recursiva pai/filho

Quando houver `questions` para uma action:

- o conjunto persistido dessa action deve ser substituído integralmente pela versão declarada em código;
- perguntas filhas devem manter referência explícita da pergunta pai;
- condições de exibição devem ser persistidas por `parent_value`.

---

## 9. CI/CD e Segurança

## 9.1 Testes

- Unit tests obrigatórios em PR
- Cenários de erro devem validar comportamento esperado (teste deve passar)

## 9.2 Trivy (padrão de esteira)

Filesystem scan:

- trivy fs --exit-code 1 --severity HIGH,CRITICAL --scanners vuln,secret --format table ./

Image scan:

- docker build --no-cache -t <image:tag> <context>
- trivy image --exit-code 1 --ignore-unfixed --severity HIGH,CRITICAL --scanners vuln --format table <image:tag>

Imagem recomendada:

- aquasec/trivy:0.69.3

---

## 10. Portabilidade por Linguagem

## 10.1 Python

- Frameworks possíveis: Django, FastAPI, Flask
- Scheduler adapters: django-q, django-rq, celery beat, APScheduler

## 10.2 Node.js

- Frameworks possíveis: NestJS, Express
- Scheduler adapters: BullMQ, Agenda, node-cron

## 10.3 Java

- Frameworks possíveis: Spring Boot
- Scheduler adapters: @Scheduled, Quartz

## 10.4 Go

- Frameworks possíveis: Gin, Fiber
- Scheduler adapters: robfig/cron

Princípio comum:

- manter os mesmos contratos de entrada/saída
- manter o mesmo modelo canônico de domínio

---

## 11. Requisitos Não Funcionais

- Observabilidade mínima:
  - logs estruturados para cada health check e execução de ação
- Idempotência em setup/sync
- Segurança:
  - sanitizar mensagens de erro antes de expor em API pública
- Performance:
  - evitar N+1 em listagem de actions e healthcare

---

## 12. Roadmap de Evolução

V1:

- Contratos e entidades canônicas
- APIs essenciais
- Scheduler + limpeza de logs
- Testes unitários e scan de segurança

V2 sugerido:

- Permissão por action
- Retry policy por serviço
- Alertas/webhooks de falhas
- Painel administrativo independente
