# API

Rotas básicas implementadas na Etapa 2 (backend):

Health
- GET /health

Tenants
- POST /api/v1/tenants
- GET /api/v1/tenants
- GET /api/v1/tenants/:id

Customers
- POST /api/v1/tenants/:tenantId/customers
- GET /api/v1/tenants/:tenantId/customers

Boletos
- POST /api/v1/tenants/:tenantId/boletos
- GET /api/v1/tenants/:tenantId/boletos
- GET /api/v1/tenants/:tenantId/boletos/:id

Formato de resposta
- Sucesso: { "data": ... }
- Erro: { "error": { "code": "...", "message": "..." } }

Exemplo criar boleto
POST /api/v1/tenants/{tenantId}/boletos
{
  "customer_id": "uuid",
  "amount_cents": 10000,
  "due_date": "2026-07-10"
}
