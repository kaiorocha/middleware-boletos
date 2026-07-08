# middleware-boletos

Plataforma para emissão e gestão de boletos com arquitetura multi-tenant.  
**Status atual:** Etapa 3 (Arquitetura de provedores) implementada com mock provider, primeiro adapter real Moncalieri Capital, factory, máquina de estados, emissão simulada, webhooks preparados e health/balance por provider.

## Stack

- Backend: Go 1.21
- Frontend: Next.js
- Banco: PostgreSQL
- Cache/futuro: Redis
- Orquestração local: Docker Compose
- CI: GitHub Actions

## Estrutura (resumo)

- `backend/cmd/api`: bootstrap da API
- `backend/internal/config`: configuração por ambiente
- `backend/internal/storage`: conexão PostgreSQL + migrations iniciais
- `backend/internal/domain`: entidades de domínio
- `backend/internal/repository`: acesso a dados
- `backend/internal/service`: regras de negócio e validações
- `backend/internal/providers`: contratos, factory, adapters, tipos, eventos e validações de integração bancária
- `frontend`: aplicação web inicial

## Rodar localmente

1. Copie variáveis:
   ```bash
   cp .env.example .env
   ```
2. Suba tudo:
   ```bash
   docker-compose up --build
   ```
3. Endpoints:
   - Backend: `http://localhost:8080`
   - Frontend: `http://localhost:3000`

## Validar com curl

```bash
curl -s http://localhost:8080/health
```

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant Demo"}'
```

## Postman Collection

A collection Postman da Etapa 3 está disponível em:

`docs/postman/middleware-boletos-etapa-3.postman_collection.json`

Ela pode ser importada no Postman para validar criação de recursos, emissão simulada, health, balance, webhook com `MockProvider` e configuração/health do provider Moncalieri sem credencial real.

## Rotas disponíveis

### Health
- `GET /health`

### Tenants
- `POST /api/v1/tenants`
- `GET /api/v1/tenants`
- `GET /api/v1/tenants/:id`

### Users
- `POST /api/v1/users`
- `GET /api/v1/users/:id`
- `GET /api/v1/tenants/:tenantId/users`

### Customers
- `POST /api/v1/tenants/:tenantId/customers`
- `GET /api/v1/tenants/:tenantId/customers`
- `GET /api/v1/tenants/:tenantId/customers/:id`
- `PUT /api/v1/tenants/:tenantId/customers/:id`

### Providers
- `POST /api/v1/tenants/:tenantId/providers`
- `GET /api/v1/tenants/:tenantId/providers`
- `GET /api/v1/tenants/:tenantId/providers/:id`
- `GET /api/v1/providers/health`
- `GET /api/v1/providers/balance`
- `POST /api/v1/providers/webhook`

### Boletos
- `POST /api/v1/tenants/:tenantId/boletos`
- `GET /api/v1/tenants/:tenantId/boletos`
- `GET /api/v1/tenants/:tenantId/boletos/:id`
- `POST /api/v1/tenants/:tenantId/boletos/:id/emit`

## Padrão de resposta

Sucesso:
```json
{ "data": {} }
```

Erro:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Descrição do erro"
  }
}
```

Duplicidade:
```json
{
  "error": {
    "code": "DUPLICATE_RESOURCE",
    "message": "Já existe um recurso com estes dados neste tenant."
  }
}
```

## Unicidade por tenant

A API bloqueia duplicidade apenas dentro do mesmo tenant, mantendo isolamento multi-tenant. A mesma informação pode existir em tenants diferentes.

- Usuários ativos não podem repetir e-mail no mesmo tenant. A comparação é case-insensitive e o e-mail é salvo normalizado com trim e lowercase.
- Clientes ativos não podem repetir documento no mesmo tenant. O documento é salvo sem máscara; documentos vazios ou nulos não bloqueiam duplicidade.
- Provedores ativos não podem repetir nome no mesmo tenant. A comparação é case-insensitive.
- Boletos ativos não podem repetir `external_id` ou `our_number` no mesmo tenant quando esses campos forem informados. Valores vazios são tratados como nulos.

Violação de unicidade retorna HTTP `409 Conflict` com `error.code = "DUPLICATE_RESOURCE"`.

## Exemplos de request

### Criar Tenant
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant A"}'
```

### Criar Customer
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/customers \
  -H "Content-Type: application/json" \
  -d '{"name":"Cliente 1","document":"12345678900"}'
```

### Criar Provider Mock
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/providers \
  -H "Content-Type: application/json" \
  -d '{"name":"Mock","config":"{\"delay_ms\":0}"}'
```

### Criar Provider Moncalieri
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/providers \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Moncalieri Capital",
    "config":"{\"base_url\":\"https://dev.moncaliericapital.com.br\",\"api_key\":\"REPLACE_WITH_SECRET\",\"codigo_canal\":0,\"codigo_cliente\":0,\"timeout_seconds\":30}"
  }'
```

### Criar e emitir Boleto com MockProvider
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/boletos \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id":"<customerId>",
    "amount_cents":15000,
    "due_date":"2026-07-30",
    "status":"CREATED"
  }'
```

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/boletos/<boletoId>/emit
```

## Observações

- A Etapa 3 mantém `MockProvider` e adiciona o adapter real `MoncalieriProvider`.
- A emissão Moncalieri exige dados completos do sacado em `types.IssueRequest.Payer`; a API da aplicação ainda precisa evoluir para buscar/enriquecer esses dados a partir de `Customer`.
- A chave `api_key` da Moncalieri nunca deve ser commitada.
- Status de boleto seguem a máquina `CREATED -> PROCESSING -> ISSUED -> PAID` ou `FAILED`/`CANCELLED`, com `PARTIAL` e `EXPIRED` após emissão.

## Testes

Testes unitários cobrem validações e regras de negócio dos services:

### Cobertura

- **Health Check** (`cmd/api/health_test.go`)
  - Verifica que `GET /health` retorna status `ok`

- **TenantService** (validações)
  - Rejeita nome vazio
  - Cria tenant válido

- **UserService** (validações)
  - Rejeita tenant_id inválido (não-UUID)
  - Rejeita email inválido
  - Rejeita email vazio
  - Cria usuário válido

- **CustomerService** (validações)
  - Rejeita tenant_id inválido (não-UUID)
  - Rejeita nome vazio
  - Cria cliente válido

- **ProviderService** (validações)
  - Rejeita tenant_id inválido (não-UUID)
  - Rejeita nome vazio
  - Cria provedor válido

- **BoletoService** (validações e emissão)
  - Rejeita tenant_id inválido
  - Rejeita customer_id inválido
  - Rejeita provider_id inválido (se fornecido)
  - Rejeita amount_cents zero
  - Rejeita amount_cents negativo
  - Rejeita due_date vazia
  - Rejeita status inválido
  - Cria boleto válido com status `CREATED`
  - Cria boleto válido com status `PROCESSING`
  - Cria boleto válido com provider opcional
  - Emite boleto via factory/provider
  - Garante idempotência quando o boleto já foi emitido

- **Providers**
  - Factory do `MockProvider`
  - Factory do `MoncalieriProvider`
  - `MockProvider` para issue, health e webhook validation
  - `MoncalieriProvider` para config, issue, get, cancel, status mapping e erros HTTP
  - Máquina de estados

### Executar testes

Dentro do container:
```bash
docker-compose up backend --build
```

Ou manualmente (requer Go 1.21):
```bash
cd backend
go test ./...
```

Esperado:
```
ok      github.com/kaiorocha/middleware-boletos/backend/cmd/api            0.008s
ok      github.com/kaiorocha/middleware-boletos/backend/internal/service   0.013s
```

### Estrutura de testes

- Testes unitários com mocks de repositories (não requerem banco de dados)
- Foco em validações de entrada e regras de negócio
- Testes de integração com banco planados para **Etapa 3+**
