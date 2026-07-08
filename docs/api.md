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
- Duplicidade: HTTP 409 com { "error": { "code": "DUPLICATE_RESOURCE", "message": "..." } }

Unicidade por tenant
- Usuários ativos não podem repetir e-mail dentro do mesmo tenant. A comparação é case-insensitive e o e-mail é persistido normalizado.
- Clientes ativos não podem repetir documento dentro do mesmo tenant. O documento é persistido sem máscara; documentos vazios ou nulos não bloqueiam duplicidade.
- Provedores ativos não podem repetir nome dentro do mesmo tenant. A comparação é case-insensitive.
- Boletos ativos não podem repetir external_id ou our_number dentro do mesmo tenant quando esses campos forem informados.
- A mesma informação pode existir em tenants diferentes.

Exemplo criar boleto
POST /api/v1/tenants/{tenantId}/boletos
{
  "customer_id": "uuid",
  "amount_cents": 10000,
  "due_date": "2026-07-10"
}
