# middleware-boletos

Plataforma para emissão e gestão de boletos com arquitetura multi-tenant.  
**Status atual:** Etapa 2 (Backend e API) implementada com persistência PostgreSQL, migrations automáticas na inicialização, services, repositories e rotas REST base.

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

## Rotas disponíveis (Etapa 2)

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

### Boletos
- `POST /api/v1/tenants/:tenantId/boletos`
- `GET /api/v1/tenants/:tenantId/boletos`
- `GET /api/v1/tenants/:tenantId/boletos/:id`

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

### Criar Provider
```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/providers \
  -H "Content-Type: application/json" \
  -d '{"name":"Banco X"}'
```

### Criar Boleto (intenção)
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

## Observações

- O boleto na Etapa 2 é apenas persistência e ciclo inicial (`CREATED`/`PENDING`).
- Emissão real com banco/provedor será implementada na **Etapa 3**.
