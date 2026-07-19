# middleware-boletos

Plataforma para emissão e gestão de boletos com arquitetura multi-tenant.  
**Status atual:** Etapa 4 (Painel Administrativo e Compliance) implementada com painel web operacional, dashboard, gestão de recursos e blacklist por tenant para bloquear emissões.

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
- `frontend`: painel administrativo web em Next.js

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
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant Demo"}'
```

## Postman Collection

A collection Postman da Etapa 4 está disponível em:

`docs/postman/middleware-boletos-etapa-4.postman_collection.json`

Ela pode ser importada no Postman para validar dashboard, blacklist, consulta de bloqueio e emissão bloqueada por compliance.

## Demo SaaS

O fluxo demonstrável da Etapa 4 está documentado em `docs/demo.md`.

Credenciais locais de desenvolvimento, quando `.env.example` é usado:

- Platform Admin: `admin@middleware.local`
- Senha: `ChangeMe123456!`

Após login, o `PLATFORM_ADMIN` cria tenants e administradores de tenant pelo painel. O `TENANT_ADMIN` acessa somente os tenants presentes no JWT, sem digitar UUID manualmente.

Em production, o login inicia com campos vazios. `NEXT_PUBLIC_DEMO_ADMIN_EMAIL` e `NEXT_PUBLIC_DEMO_ADMIN_PASSWORD` são variáveis públicas apenas para desenvolvimento/demo local e nunca devem carregar senha real.

Bootstrap automático do `PLATFORM_ADMIN` só roda livremente em `APP_ENV=development`. Em `APP_ENV=production`, exige `ENABLE_ADMIN_BOOTSTRAP=true`; depois da primeira criação, volte para `false` e remova `BOOTSTRAP_ADMIN_PASSWORD`.

## Rotas disponíveis

### Health
- `GET /health`

### Tenants
- `POST /api/v1/tenants`
- `GET /api/v1/tenants`
- `GET /api/v1/tenants/:id`
- `GET /api/v1/me/tenants`
- `POST /api/v1/admin/tenants`

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

### Dashboard
- `GET /api/v1/tenants/:tenantId/dashboard`

### Compliance
- `POST /api/v1/tenants/:tenantId/blacklist`
- `GET /api/v1/tenants/:tenantId/blacklist`
- `GET /api/v1/tenants/:tenantId/blacklist/check?document=...`
- `GET /api/v1/tenants/:tenantId/blacklist/:id`
- `PUT /api/v1/tenants/:tenantId/blacklist/:id`
- `DELETE /api/v1/tenants/:tenantId/blacklist/:id`
- `POST /api/v1/tenants/:tenantId/blacklist/:id/block`
- `POST /api/v1/tenants/:tenantId/blacklist/:id/unblock`

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

## Compliance: blacklist de emissão

Cada tenant possui sua própria lista de CPF/CNPJ bloqueados. Antes de chamar qualquer provider bancário, o `BoletoService.Emit` consulta a blacklist usando o documento do customer.

Se o documento estiver bloqueado, a API interrompe a emissão e retorna HTTP `409 Conflict`:

```json
{
  "error": {
    "code": "CUSTOMER_BLOCKED",
    "message": "Este cliente está bloqueado para novas emissões."
  }
}
```

O bloqueio é aplicado no backend, no fluxo central de emissão, para evitar bypass por painel, API externa ou integrações futuras.

## Autorização multi-tenant

Rotas administrativas e operacionais exigem `Authorization: Bearer <JWT>`. A exceção pública é `GET /health`.

O JWT deve conter `sub`, `tenant_id` ou `tenant_ids`, e pode conter `roles`. Rotas tenant-scoped validam o tenant da URL contra os tenants presentes na identidade autenticada. Sem token ou com token inválido, a API retorna HTTP `401` com `UNAUTHORIZED`. Se o usuário autenticado não tiver acesso ao tenant da URL, retorna HTTP `403` com `FORBIDDEN`.

Headers arbitrários como `X-User-ID`, `X-Tenant-ID` e `X-Tenant-IDs` não autenticam usuários em produção. Em desenvolvimento local, somente com `APP_ENV=development`, é possível usar `X-Dev-User-ID` e `X-Dev-Tenant-ID`.

`GET /api/v1/tenants` e `POST /api/v1/tenants` exigem role `PLATFORM_ADMIN`. Usuários comuns obtêm seus tenants por `GET /api/v1/me/tenants` ou diretamente da sessão/JWT.

Em produção, `JWT_SECRET`, `JWT_ISSUER` e `JWT_AUDIENCE` são obrigatórios; `JWT_SECRET` deve ter pelo menos 32 caracteres. Configuração inválida encerra o startup com `auth config invalid`.

Detalhes: `docs/authentication.md`.

## Exemplos de request

### Criar Tenant
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant A"}'
```

### Criar Customer
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/customers \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Cliente 1",
    "document":"12345678900",
    "email":"cliente@example.com",
    "address":"Rua Um",
    "number":"123",
    "complement":"Apto 4",
    "district":"Centro",
    "city":"Sao Paulo",
    "state":"SP",
    "postal_code":"12345-678"
  }'
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
- A emissão Moncalieri exige dados completos do sacado. O `BoletoService` busca o `Customer` e usa `DefaultPayerBuilder` para montar `types.IssueRequest.Payer`.
- Toda regra de tradução `Customer -> Payer` fica em `backend/internal/providers/base/payer_builder.go`.
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
  - `DefaultPayerBuilder` para normalização e validação de pagador
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
