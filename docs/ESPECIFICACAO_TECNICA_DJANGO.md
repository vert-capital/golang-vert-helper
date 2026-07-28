# Django Vert Helper - Especificação Técnica V1

## 📋 Visão Geral

Esta biblioteca Django oferece um sistema integrado para:
- **Monitoramento de Saúde** de serviços (PostgreSQL, S3, Kafka)
- **Sistema de Ações** registradas com formulários condicionais
- **APIs RESTful** para consumo de dados de saúde e execução de ações
- **Verificação de Saúde da Aplicação** sem dependência do Django (Docker)

---

## 📊 Arquitetura de Banco de Dados

### 1. Tabela `Service`
Registra os serviços disponíveis para monitoramento.

```
Service
├── id (UUID, PK)
├── name (CharField, unique) - "S3", "POSTGRESQL", "KAFKA"
├── is_active (BooleanField, default=True) - Soft delete
├── created_at (DateTimeField, auto_now_add)
└── updated_at (DateTimeField, auto_now)

Índices: (name), (is_active)
```

**Django Admin:** ✓ Disponível (filtrado por is_active=True por default)

**Soft Delete:** Quando um serviço é removido do settings, é marcado como `is_active=False`. Não aparece nas APIs, mas mantém histórico em `ServiceHealth`.

---

### 2. Tabela `ServiceHealth`
Histórico de verificações de saúde dos serviços.

```
ServiceHealth
├── id (UUID, PK)
├── service_id (ForeignKey → Service)
├── status (CharField) - "OK", "FAILED", "UNKNOWN"
├── message (TextField, nullable) - "Connection timeout"
├── checked_at (DateTimeField) - Momento do check
└── created_at (DateTimeField, auto_now_add)

Índices: (service_id, checked_at DESC), (status)
```

**Django Admin:** ✓ Disponível

**Limpeza:** Task agendada (Django Q ou Django RQ) para remover registros antigos

**API Default:** Retorna **última consulta** de cada serviço

---

### 3. Tabela `Action`
Registro de ações disponíveis no sistema.

```
Action
├── id (UUID, PK)
├── slug (CharField, unique) - "execute-without-kafka"
├── name (CharField) - "Executar sem Kafka"
├── description (TextField)
├── services (ArrayField de UUID ForeignKey → Service)
├── function_path (CharField) - "app.actions.execute_without_kafka"
├── status (CharField) - "active", "inactive"
├── created_at (DateTimeField, auto_now_add)
├── updated_at (DateTimeField, auto_now)
└── metadata (JSONField, nullable) - Dados extras

Índices: (slug), (status), (services)
```

**Django Admin:** ✓ Disponível

**Sincronização:** 
- Usa Management Command `vert_helper_setup` ou `vert_helper_sync_actions`
- Descobre actions com decorator `@helper_action`
- **Deleta actions orfãs** (que não existem mais no código)

---

### 4. Tabela `Question`
Formulários condicionais para as ações.

```
Question
├── id (UUID, PK)
├── action_id (ForeignKey → Action)
├── label (CharField) - "O arquivo é CSV ou JSON?"
├── type (CharField) - "radio", "text", "textarea", "file", "select"
├── options (JSONField, nullable) - ["CSV", "JSON"]
├── is_required (BooleanField, default=True)
├── parent_question (ForeignKey → Question, nullable)
├── parent_value (CharField, nullable) - Valor da resposta que ativa esta pergunta
├── action_kwarg (CharField, nullable) - Nome do parâmetro da função (ex: "workflow_id")
├── is_first (BooleanField) - True apenas para primeira pergunta
├── created_at (DateTimeField, auto_now_add)
└── updated_at (DateTimeField, auto_now)

Índices: (action_id, is_first), (parent_question, parent_value)
```

**Django Admin:** ✓ Disponível

**Fluxo Condicional:**
1. Busca perguntas com `is_first=True` para iniciar
2. Ao responder, busca: `where parent_question = <id> AND parent_value = <resposta>`
3. Se não houver próxima pergunta, finaliza o formulário

---

### 5. Tabela `ActionExecution` (Auditoria/Histórico)
Registro de execuções de ações.

```
ActionExecution
├── id (UUID, PK)
├── action_id (ForeignKey → Action)
├── responses (JSONField) - {"1": "CSV", "2": "Arquivo", "3": "workflow-123"}
├── result (JSONField) - {"status": "success|error|info", "message": "..."}
├── executed_by (ForeignKey → User, nullable)
├── executed_at (DateTimeField, auto_now_add)
└── created_at (DateTimeField, auto_now_add)

Índices: (action_id, executed_at DESC), (executed_by)
```

**Django Admin:** ✓ Disponível (read-only)

---

## ⚙️ Configuração via Settings

```python
VERT_HELPER = {
    # Autenticação
    "PERMISSION_CLASS": "rest_framework.permissions.AllowAny",  # padrão
    
    # Agendamento de Health Checks
    "SCHEDULER": "django_q",  # ou "rq" ou None (não agenda)
    "HEALTH_CHECK_INTERVAL": 600,  # segundos (padrão: 10 min)
    "HEALTH_CHECK_AUTO_REGISTER": True,
    
    # Serviços a Monitorar
    "SERVICES": {
        "postgres": {
            "label": "PostgreSQL",
            "enabled": True,
            "function": "vert_helper.health_checks.postgres.check_postgres",
            "context": {
                "host": "localhost",
                "port": 5432,
                "database": "mydb",
                "user": "postgres",
                "password": "password"
            }
        },
        "s3": {
            "label": "AWS S3",
            "enabled": True,
            "function": "vert_helper.health_checks.s3.check_s3",
            "context": {
                "aws_access_key_id": "...",
                "aws_secret_access_key": "...",
                "bucket_name": "..."
            }
        },
        "kafka": {
            "label": "Apache Kafka",
            "enabled": True,
            "function": "vert_helper.health_checks.kafka.check_kafka",
            "context": {
                "bootstrap_servers": ["localhost:9092"],
                "timeout": 5
            }
        }
    }
}
```

**Comportamento:**
- Se `SCHEDULER` está preenchido: registra task com interval = `HEALTH_CHECK_INTERVAL` (ou 600)
- Se `SCHEDULER` é `None` ou vazio: **não agenda** automaticamente
- Health checks sob demanda via API ainda funcionam

---

## 🔌 APIs

### 1. Healthcare Status
**Endpoint:** `GET /api/helper/v1/healthcare/`

**Autenticação:** Conforme `VERT_HELPER["PERMISSION_CLASS"]`

**Response:**
```json
{
    "S3": {
        "status": "OK",
        "last_updated": "2024-06-10T12:34:56Z"
    },
    "POSTGRESQL": {
        "status": "FAILED",
        "message": "Connection timeout",
        "last_updated": "2024-06-10T12:35:00Z"
    },
    "KAFKA": {
        "status": "UNKNOWN",
        "message": "Service not configured",
        "last_updated": null
    }
}
```

**Opção:** Query param `?force_refresh=true` para forçar check imediato

---

### 2. Listar Actions
**Endpoint:** `GET /api/helper/v1/actions/`

**Autenticação:** Conforme `VERT_HELPER["PERMISSION_CLASS"]`

**Query Params:**
- `service=<service_name>` - Filtrar por serviço
- `search=<termo>` - Buscar por nome/descrição
- `page=<num>` - Paginação
- `page_size=<num>` - Itens por página (padrão: 10)
- `ordering=-name` - Ordenar por campo

**Response:**
```json
{
    "count": 15,
    "next": "http://api.example.com/helper/v1/actions/?page=2",
    "previous": null,
    "results": [
        {
            "id": "uuid-123",
            "slug": "execute-without-kafka",
            "name": "Executar sem Kafka",
            "description": "Executa operação sem dependência do Kafka",
            "services": ["S3", "KAFKA"],
            "status": "active",
            "is_recommended": true,  // se serviço falhou
            "created_at": "2024-06-01T10:00:00Z"
        }
    ]
}
```

**Ordenação Automática:** Se algum serviço em `services` está `FAILED`, action aparece com `is_recommended: true` no topo

---

### 3. Detalhes da Action + Formulário
**Endpoint:** `GET /api/helper/v1/actions/<slug>/`

**Autenticação:** Conforme `VERT_HELPER["PERMISSION_CLASS"]`

**Response:**
```json
{
    "id": "uuid-123",
    "slug": "generate-document",
    "name": "Gerar Documento",
    "description": "Gera documento em CSV ou JSON",
    "services": ["S3"],
    "status": "active",
    "questions": [
        {
            "id": "q1",
            "label": "O arquivo é CSV ou JSON?",
            "type": "radio",
            "options": ["CSV", "JSON"],
            "is_required": true,
            "parent_question": null,
            "parent_value": null,
            "action_kwarg": "file_type"
        },
        {
            "id": "q2",
            "label": "Você irá mandar o arquivo ou URL?",
            "type": "radio",
            "options": ["Arquivo", "URL"],
            "is_required": true,
            "parent_question": "q1",
            "parent_value": "CSV",
            "action_kwarg": "csv_source"
        },
        {
            "id": "q3",
            "label": "Qual ID do workflow?",
            "type": "text",
            "options": null,
            "is_required": true,
            "parent_question": "q1",
            "parent_value": "JSON",
            "action_kwarg": "workflow_id"
        }
    ],
    "created_at": "2024-06-01T10:00:00Z"
}
```

---

### 4. Executar Action
**Endpoint:** `POST /api/helper/v1/actions/<slug>/execute/`

**Autenticação:** Conforme `VERT_HELPER["PERMISSION_CLASS"]`

**Request Body:**
```json
{
    "questions": {
        "q1": "CSV",
        "q2": "Arquivo",
        "q3": "workflow-123"
    }
}
```

**Response (Sucesso):**
```json
{
    "status": "success",
    "message": "Documento gerado com sucesso",
    "data": {
        "document_id": "doc-456",
        "url": "s3://bucket/document-456.csv"
    }
}
```

**Response (Erro):**
```json
{
    "status": "error",
    "message": "Falha ao gerar documento",
    "details": "S3 connection failed"
}
```

**Response (Info):**
```json
{
    "status": "info",
    "message": "Documento não pode ser gerado no momento",
    "steps": [
        "Verifique se o workflow 123 está ativo",
        "Confirme se o bucket S3 está acessível",
        "Tente novamente em alguns minutos"
    ]
}
```

---

### 5. App Health (Sem Django)
**Endpoint:** `GET /api/helper/v1/app-health/`

**Autenticação:** ❌ Nenhuma

**Response:**
```json
{
    "status": "stable"
}
```

ou

```json
{
    "status": "failed",
    "message": "Application startup failed"
}
```

**Implementação:** Arquivo JSON estático (`/app/health.json`) atualizado via cronjob no Docker

---

## 🎯 Management Commands

### 1. Setup Completo (Recomendado no Deploy)

**Comando:**
```bash
python manage.py vert_helper_setup
```

**Executa:**
1. **Sincroniza Services**
   - Lê todos os serviços em `VERT_HELPER["SERVICES"]` (nativos + custom)
   - Cria novos serviços na tabela `Service`
   - Ativa (`is_active=True`) serviços já existentes
   - Soft delete (`is_active=False`) serviços removidos do settings
   - Mantém histórico em `ServiceHealth` intacto

2. **Sincroniza Actions**
   - Descobre todas as funções com `@helper_action`
   - Cria registros em `Action`
   - Atualiza existentes
   - Deleta orfãs (que não existem mais no código)

3. **Registra Task Agendada (opcional)**
   - Se `VERT_HELPER["SCHEDULER"]` está configurado:
     - Django Q: registra job `vert_helper.tasks.run_health_checks`
     - Django RQ: cria job recorrente

**Output Exemplo:**
```
✓ Sincronizando Services...
  ✓ Criado: PostgreSQL
  ✓ Criado: S3
  ✓ Criado: Kafka
  ✓ Criado: Custom API (custom)
  ✓ Desativado: Old Service
  
✓ Sincronizando Actions...
  ✓ Criado: generate-report
  ✓ Atualizado: execute-without-kafka
  ✓ Deletado: deprecated-action
  
✓ Registrando task agendada...
  ✓ Health check agendado a cada 600s (Django Q)
  
Setup concluído com sucesso!
```

---

### 2. Sincronizar Apenas Actions

**Comando:**
```bash
python manage.py vert_helper_sync_actions
```

**Executa:**
- Mesmo que o passo 2 do setup (sem sincronizar services)
- Útil quando apenas adiciona novas ações sem mudar serviços

---

## 🎯 Decorator `@helper_action`

**Uso:**
```python
from vert_helper import helper_action

@helper_action(
    slug="execute-without-kafka",
    name="Executar sem Kafka",
    description="Executa operação sem dependência do Kafka",
    services=["S3", "KAFKA"],
    questions=[
        {
            "label": "Tipo do arquivo",
            "type": "radio",
            "options": ["CSV", "JSON"],
            "is_required": True,
            "action_kwarg": "file_type",
            "children": [
                {
                    "label": "Fonte do CSV",
                    "type": "radio",
                    "options": ["Arquivo", "URL"],
                    "is_required": True,
                    "action_kwarg": "csv_source",
                    "parent_value": "CSV"
                },
                {
                    "label": "ID do workflow",
                    "type": "text",
                    "is_required": True,
                    "action_kwarg": "workflow_id",
                    "parent_value": "JSON"
                }
            ]
        }
    ]
)
def execute_without_kafka(responses):
    """
    Args:
        responses: Dict com IDs de perguntas e respostas
                  {"q1": "CSV", "q2": "Arquivo"}
    
    Returns:
        Dict com estrutura padrão:
        {
            "status": "success|error|info",
            "message": "...",
            "data": {...},  # opcional
            "steps": [...]  # opcional (apenas para info)
        }
    """
    file_type = responses.get("q1")
    
    if file_type == "CSV":
        return {
            "status": "success",
            "message": "Arquivo CSV processado",
            "data": {"file_id": "123"}
        }
    
    return {
        "status": "error",
        "message": "Tipo de arquivo não suportado"
    }
```

**Management Command:**
```bash
python manage.py vert_helper_sync_actions
```

Executa:
1. Descobre todas as funções decoradas com `@helper_action`
2. Cria/atualiza registros em `Action`
3. **Deleta actions orfãs** (que não existem mais no código)
4. Sincroniza estrutura de perguntas a partir do campo `questions` do decorator

### Estrutura de `questions` no decorator

Cada item de `questions` representa uma pergunta raiz e aceita os campos:

- `label` (obrigatório)
- `type` (obrigatório): `radio`, `text`, `textarea`, `file`, `select`
- `options` (opcional): lista de opções
- `is_required` (opcional, padrão `False`)
- `action_kwarg` (opcional): nome do argumento final para o handler
- `children` (opcional): lista de perguntas filhas

Cada item em `children` segue a mesma estrutura e pode incluir:

- `parent_value` (opcional): valor da resposta da pergunta pai que habilita a filha

### Regra de sincronização das perguntas

Na sincronização de actions:

- se a action possuir `questions`, as perguntas persistidas da action são removidas e recriadas conforme a árvore declarada;
- a relação pai/filho é montada recursivamente via `children`;
- perguntas filhas condicionais devem informar `parent_value` para ativação por resposta.

---

## 🐳 Docker - App Health

**Implementação:** Cronjob + Script Bash

**Arquivo:** `/app/health_check.sh`
```bash
#!/bin/bash
set -e

HEALTH_STATUS="stable"

# Teste se Django está respondendo
if ! curl -s http://localhost:8000/api/helper/v1/healthcare/ > /dev/null 2>&1; then
    HEALTH_STATUS="failed"
fi

cat > /app/health.json << EOF
{
  "status": "$HEALTH_STATUS",
  "timestamp": "$(TZ=America/Sao_Paulo date +"%Y-%m-%dT%H:%M:%S%:z")"
}
EOF
```

**Dockerfile (adições):**
```dockerfile
# ... resto do Dockerfile ...

RUN chmod +x /app/health_check.sh

# Instalar cron
RUN apt-get update && apt-get install -y cron && rm -rf /var/lib/apt/lists/*

# Registrar cronjob
RUN echo "*/10 * * * * /app/health_check.sh" | crontab -

# Servir health.json via nginx
# (já configurado no seu reverse proxy)

# Iniciar cron + aplicação
CMD service cron start && gunicorn ...
```

**Nginx Config:**
```nginx
location /api/helper/v1/app-health/ {
    alias /app/health.json;
}
```

---

## 📝 Checklist de Implementação

- [ ] Models: Service, ServiceHealth, Action, Question, ActionExecution
- [ ] Serializers: ServiceHealthSerializer, ActionSerializer, QuestionSerializer
- [ ] Views: HealthcareViewSet, ActionViewSet, ActionExecuteView
- [ ] Health checks functions: postgres, s3, kafka
- [ ] Task/Job agendado para health checks (Django Q ou RQ)
- [ ] Management command: `vert_helper_sync_actions`
- [ ] Decorator: `@helper_action`
- [ ] Docker integration: cronjob + health.json
- [ ] Django Admin config para todas as tabelas
- [ ] Testes unitários e de integração
- [ ] Documentação de uso (manual do usuário)

---

## 🚀 Deployment

1. **Deploy da Aplicação:**
   ```bash
   python manage.py migrate
   python manage.py vert_helper_sync_actions
   ```

2. **Configurar Scheduler (opcional):**
   - Se `SCHEDULER = "django_q"`: `qcluster` deve estar rodando
   - Se `SCHEDULER = "rq"`: Worker RQ deve estar em background

3. **Docker:**
   - Incluir `health_check.sh`
   - Configurar cronjob
   - Servir `health.json` via reverse proxy

---

## 📚 Próximas Etapas (V2)

- [ ] Banco de dados persistente para execuções (ActionExecution já preparada)
- [ ] Permissões por ação
- [ ] Webhooks em caso de falha de serviço
- [ ] Alertas customizados
- [ ] UI para gerenciar actions e formulários

