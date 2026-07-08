# API — Etapa 2

Documentação das rotas REST implementadas na **Etapa 2 — Backend e API**.

Nesta etapa, a API foi preparada para persistência das principais entidades da plataforma, com PostgreSQL, services, repositories, validações básicas e padrão de resposta JSON.

> Observação: a emissão real de boletos com bancos/provedores ainda não faz parte desta etapa. O boleto criado aqui representa uma intenção de emissão, com status inicial `CREATED` ou `PENDING`.

## Base URL local

```text
http://localhost:8080
```

## Padrão de resposta

### Sucesso

```json
{
  "data": {}
}
```

### Erro

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Descrição do erro"
  }
}
```

## Health Check

### GET /health

Valida se a API está disponível.

```bash
curl -s http://localhost:8080/health
```

Resposta esperada:

```json
{
  "data": {
    "status": "ok"
  }
}
```

## Tenants

### POST /api/v1/tenants

Cria um tenant/empresa.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant A"}'
```

### GET /api/v1/tenants

Lista tenants cadastrados.

```bash
curl -s http://localhost:8080/api/v1/tenants
```

### GET /api/v1/tenants/:id

Busca um tenant por ID.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>
```

## Users

### POST /api/v1/users

Cria um usuário vinculado a um tenant.

```bash
curl -s -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id":"<tenantId>",
    "email":"usuario@empresa.com",
    "name":"Usuário Demo"
  }'
```

### GET /api/v1/users/:id

Busca um usuário por ID.

```bash
curl -s http://localhost:8080/api/v1/users/<userId>
```

### GET /api/v1/tenants/:tenantId/users

Lista usuários por tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/users
```

## Customers

### POST /api/v1/tenants/:tenantId/customers

Cria um cliente/sacado vinculado a um tenant.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/customers \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Cliente 1",
    "document":"12345678900"
  }'
```

### GET /api/v1/tenants/:tenantId/customers

Lista clientes por tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/customers
```

### GET /api/v1/tenants/:tenantId/customers/:id

Busca cliente por ID dentro do tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/customers/<customerId>
```

### PUT /api/v1/tenants/:tenantId/customers/:id

Atualiza dados básicos de um cliente.

```bash
curl -s -X PUT http://localhost:8080/api/v1/tenants/<tenantId>/customers/<customerId> \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Cliente 1 Atualizado",
    "document":"12345678900"
  }'
```

## Providers

### POST /api/v1/tenants/:tenantId/providers

Cria um provedor bancário/gateway vinculado ao tenant.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/providers \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Banco X",
    "config":"{}"
  }'
```

### GET /api/v1/tenants/:tenantId/providers

Lista provedores por tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/providers
```

### GET /api/v1/tenants/:tenantId/providers/:id

Busca provedor por ID dentro do tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/providers/<providerId>
```

## Boletos

### POST /api/v1/tenants/:tenantId/boletos

Cria uma intenção de boleto. A emissão real com banco/provedor será implementada na Etapa 3.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/boletos \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id":"<customerId>",
    "provider_id":"<providerId>",
    "amount_cents":15000,
    "due_date":"2026-07-30",
    "status":"CREATED"
  }'
```

Campos principais:

| Campo | Obrigatório | Observação |
|---|---:|---|
| `customer_id` | Sim | UUID do cliente/sacado. |
| `provider_id` | Não | UUID do provedor. Opcional nesta etapa. |
| `amount_cents` | Sim | Valor em centavos. Deve ser maior que zero. |
| `due_date` | Sim | Data no formato `YYYY-MM-DD`. |
| `status` | Não | Aceita `CREATED` ou `PENDING`. Se vazio, assume `CREATED`. |

### GET /api/v1/tenants/:tenantId/boletos

Lista boletos por tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/boletos
```

### GET /api/v1/tenants/:tenantId/boletos/:id

Busca boleto por ID dentro do tenant.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/boletos/<boletoId>
```

## Validações implementadas

- UUID válido para parâmetros e campos relacionais.
- Nome obrigatório para tenant, customer e provider.
- E-mail válido para user.
- Valor do boleto maior que zero.
- Vencimento obrigatório para boleto.
- Status de boleto restrito a `CREATED` ou `PENDING`.

## Limitações conhecidas da Etapa 2

- Ainda não há autenticação/autorização real.
- Ainda não há integração bancária real.
- Ainda não há emissão, cancelamento ou consulta real de boleto junto a provedor.
- Webhooks reais ficam para a etapa de integração bancária.
