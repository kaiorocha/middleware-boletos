# API — Etapa 4

Documentação das rotas REST implementadas até a **Etapa 4 — Painel Administrativo e Compliance**.

Nesta etapa, a API foi preparada para persistência das principais entidades da plataforma, com PostgreSQL, services, repositories, validações básicas e padrão de resposta JSON.

> Observação: a Etapa 4 adiciona o painel administrativo web e o módulo de Compliance/Blacklist. A emissão continua usando `MockProvider` e `MoncalieriProvider`, mas agora consulta a blacklist antes de chamar qualquer provider.

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

## Unicidade por tenant

A API garante isolamento entre tenants. Recursos podem existir com os mesmos dados em tenants diferentes, porém não podem ser duplicados dentro do mesmo tenant.

Regras implementadas:

- Usuários ativos não podem repetir e-mail no mesmo tenant.
- Clientes ativos não podem repetir documento no mesmo tenant.
- Provedores ativos não podem repetir nome no mesmo tenant.
- Boletos ativos não podem repetir `external_id` nem `our_number` quando esses campos forem informados.

Os seguintes campos são normalizados antes da persistência:

- email: trim + lowercase
- documento: somente números
- nome do provider: trim
- external_id: trim; string vazia vira null
- our_number: trim; string vazia vira null

Quando ocorre violação de unicidade, a API retorna:

HTTP `409 Conflict`

```json
{
  "error": {
    "code": "DUPLICATE_RESOURCE",
    "message": "Descrição específica da duplicidade."
  }
}
```

## Compliance: blacklist por tenant

Cada tenant mantém uma lista isolada de documentos bloqueados para novas emissões. O mesmo CPF/CNPJ pode estar bloqueado em um tenant e liberado em outro.

Antes de chamar o provider bancário, `BoletoService.Emit` busca o customer do boleto e consulta a blacklist usando o documento normalizado. Se houver bloqueio ativo, o fluxo é interrompido antes de `PayerBuilder`, `ProviderFactory` e `IssueBoleto`.

Resposta para emissão bloqueada:

HTTP `409 Conflict`

```json
{
  "error": {
    "code": "CUSTOMER_BLOCKED",
    "message": "Este cliente está bloqueado para novas emissões."
  }
}
```

Eventos de compliance são registrados em `audit_logs` para bloqueio, desbloqueio e tentativa de emissão bloqueada.

## Autorização multi-tenant

Todas as rotas administrativas e operacionais exigem autenticação JWT:

```http
Authorization: Bearer <JWT>
```

Exceção pública:

- `GET /health`

O JWT deve conter `sub` e `tenant_id` ou `tenant_ids`. Todas as rotas tenant-scoped validam que a identidade autenticada pode operar o tenant solicitado na URL.

Respostas de autorização:

- HTTP `401` com `UNAUTHORIZED` quando não houver identidade autenticada.
- HTTP `403` com `FORBIDDEN` quando o usuário não puder operar o tenant solicitado.

Em desenvolvimento explícito (`APP_ENV=development`), `X-Dev-User-ID` e `X-Dev-Tenant-ID` podem ser usados para operação local controlada. Esses headers são ignorados em produção.

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

## Dashboard

### GET /api/v1/tenants/:tenantId/dashboard

Retorna indicadores operacionais do tenant, com filtros opcionais `from` e `to` no formato `YYYY-MM-DD`.

```bash
curl -s "http://localhost:8080/api/v1/tenants/<tenantId>/dashboard?from=2026-07-01&to=2026-07-31"
```

Indicadores retornados em `data`:

- `total_boletos`
- `boletos_emitidos`
- `boletos_em_processamento`
- `boletos_pagos`
- `boletos_vencidos`
- `boletos_cancelados`
- `boletos_com_falha`
- `valor_total_emitido`
- `by_status`

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

Campos opcionais aceitos em customer:

| Campo | Observação |
|---|---|
| `email` | Validado quando informado. |
| `address` | Obrigatório para emissão real via providers que exigem pagador. |
| `number` | Usado na composição do endereço completo. |
| `complement` | Usado na composição do endereço completo. |
| `district` | Obrigatório para emissão real. |
| `city` | Obrigatório para emissão real. |
| `state` | UF com 2 caracteres; normalizada para uppercase. |
| `postal_code` | CEP; máscara removida e validado com 8 dígitos quando informado. |

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
    "name":"Mock",
    "config":"{\"delay_ms\":0}"
  }'
```

Exemplo para Moncalieri Capital, sem credencial real:

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/providers \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Moncalieri Capital",
    "config":"{\"base_url\":\"https://dev.moncaliericapital.com.br\",\"api_key\":\"REPLACE_WITH_SECRET\",\"codigo_canal\":0,\"codigo_cliente\":0,\"timeout_seconds\":30,\"instrucoes\":\"Pagar ate o vencimento.\"}"
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

### GET /api/v1/providers/health

Consulta o health do provider. Sem parâmetros, usa `MockProvider`; com `tenant_id` e `provider_id`, carrega o provider persistido.

```bash
curl -s "http://localhost:8080/api/v1/providers/health?tenant_id=<tenantId>&provider_id=<providerId>"
```

### GET /api/v1/providers/balance

Consulta saldo padronizado do provider.

Para Moncalieri, retorna `UNSUPPORTED_OPERATION`, pois a especificação enviada não possui endpoint de saldo.

```bash
curl -s "http://localhost:8080/api/v1/providers/balance?tenant_id=<tenantId>&provider_id=<providerId>"
```

### POST /api/v1/providers/webhook

Recebe, valida e converte webhook do provider.

Para Moncalieri, retorna `UNSUPPORTED_OPERATION`, pois a especificação enviada não descreve webhooks.

```bash
curl -s -X POST "http://localhost:8080/api/v1/providers/webhook?tenant_id=<tenantId>&provider_id=<providerId>" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"boleto.paid",
    "external_id":"mock-ext",
    "our_number":"MOCK123",
    "status":"PAID"
  }'
```

## Boletos

### POST /api/v1/tenants/:tenantId/boletos

Cria uma intenção de boleto.

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
| `status` | Não | Se vazio, assume `CREATED`. |

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

### POST /api/v1/tenants/:tenantId/boletos/:id/emit

Emite o boleto usando o provider vinculado ao boleto.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/boletos/<boletoId>/emit \
  -H "X-Request-ID: req-demo-1"
```

Campos persistidos após emissão:

| Campo | Origem |
|---|---|
| `status` | Retorno padronizado do provider (`ISSUED` no mock) |
| `external_id` | Identificador externo fake |
| `barcode` | Código de barras fake |
| `digitable_line` | Linha digitável fake |
| `our_number` | Nosso número fake |
| `issued_at` | Timestamp da emissão simulada |

Para Moncalieri, o adapter real mapeia `Data.NossoNumero`, `Data.LinhaDigitavel` e `Data.CodigoBarras` da API do provider. A chamada exige dados completos do sacado no customer (`document`, `name`, endereço, bairro, cidade, CEP e UF). Se faltar algum campo obrigatório, a API retorna `INVALID_PAYER`.

## Compliance

### POST /api/v1/tenants/:tenantId/blacklist

Cria um bloqueio de documento no tenant.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/blacklist \
  -H "Content-Type: application/json" \
  -d '{
    "document":"123.456.789-00",
    "name":"Cliente Bloqueado",
    "reason":"Solicitação do cliente",
    "notes":"Não enviar novas cobranças.",
    "source":"MANUAL"
  }'
```

Campos:

| Campo | Observação |
|---|---|
| `document` | Obrigatório; normalizado para somente números. |
| `name` | Nome de referência. |
| `reason` | Motivo exibido em consultas de bloqueio. |
| `notes` | Observações internas. |
| `source` | `MANUAL`, `API` ou `IMPORT`; assume `MANUAL` se vazio. |

Duplicidade ativa de documento no mesmo tenant retorna `DUPLICATE_RESOURCE`.

### GET /api/v1/tenants/:tenantId/blacklist

Lista bloqueios do tenant. Aceita filtros `q` e `active`.

```bash
curl -s "http://localhost:8080/api/v1/tenants/<tenantId>/blacklist?q=12345678900&active=true"
```

### GET /api/v1/tenants/:tenantId/blacklist/check?document=...

Consulta otimizada para saber se um documento está bloqueado.

```bash
curl -s "http://localhost:8080/api/v1/tenants/<tenantId>/blacklist/check?document=123.456.789-00"
```

Resposta quando bloqueado:

```json
{
  "blocked": true,
  "reason": "Solicitação do cliente"
}
```

Resposta quando liberado:

```json
{
  "blocked": false
}
```

### GET /api/v1/tenants/:tenantId/blacklist/:id

Busca um bloqueio por ID dentro do tenant.

### PUT /api/v1/tenants/:tenantId/blacklist/:id

Atualiza dados do bloqueio.

### DELETE /api/v1/tenants/:tenantId/blacklist/:id

Exclui logicamente o bloqueio.

### POST /api/v1/tenants/:tenantId/blacklist/:id/block

Reativa um bloqueio.

### POST /api/v1/tenants/:tenantId/blacklist/:id/unblock

Desativa um bloqueio sem excluir o registro.

## Validações implementadas

- UUID válido para parâmetros e campos relacionais.
- Nome obrigatório para tenant, customer e provider.
- E-mail válido para user.
- Valor do boleto maior que zero.
- Vencimento obrigatório para boleto.
- Status de boleto restrito aos estados padronizados.
- Transições de boleto validadas pela máquina de estados.
- Customer valida email, documento, UF e CEP quando esses campos são informados.
- Emissão bloqueada automaticamente para documentos ativos na blacklist do tenant.

## Limitações conhecidas

- Ainda não há autenticação/autorização real.
- Cancelamento e consulta real junto a provedores reais ficam para etapas posteriores.
