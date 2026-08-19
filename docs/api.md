# API — Etapa 5

Documentação das rotas REST implementadas até a **Etapa 5 — Boleto Proposta, Compliance por email e Production Readiness**.

Nesta etapa, o fluxo principal passa a ser boleto proposta: a API cria uma intenção de boleto com email do destinatário, valor, vencimento, provider autorizado para o tenant e `external_id` opcional. A emissão consulta Compliance por email antes de chamar o provider.

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

Cada tenant mantém uma lista isolada de bloqueios para novas emissões. O fluxo principal da Etapa 5 é bloqueio por email do destinatário:

```json
{
  "entry_type": "EMAIL",
  "value": "cliente@example.com",
  "reason": "Cliente solicitou não receber novas emissões",
  "source": "MANUAL"
}
```

Antes de chamar o provider bancário em boletos proposta, `BoletoService.Emit` consulta a blacklist usando o email normalizado (`trim + lowercase`). Se houver bloqueio ativo, o fluxo é interrompido antes de `ProviderFactory` e `IssueBoleto`.

Resposta para emissão bloqueada:

HTTP `409 Conflict`

```json
{
  "error": {
    "code": "RECIPIENT_BLOCKED",
    "message": "Este destinatário está bloqueado para novas emissões."
  }
}
```

O fluxo por documento (`entry_type = DOCUMENT`) continua suportado para boletos tradicionais com `customer_id` e retorna `CUSTOMER_BLOCKED` quando o documento do customer estiver bloqueado.

Eventos de compliance são registrados em `audit_logs` para bloqueio, desbloqueio e tentativa de emissão bloqueada.

## Autorização multi-tenant

Todas as rotas administrativas e operacionais exigem autenticação JWT:

```http
Authorization: Bearer <JWT>
```

Exceções públicas:

- `GET /health`
- `GET /ready`

O JWT deve conter `sub` e `tenant_id` ou `tenant_ids`. A claim `roles` é usada para RBAC. Todas as rotas tenant-scoped validam que a identidade autenticada pode operar o tenant solicitado na URL.

Respostas de autorização:

- HTTP `401` com `UNAUTHORIZED` quando não houver identidade autenticada.
- HTTP `403` com `FORBIDDEN` quando o usuário não puder operar o tenant solicitado.

Em desenvolvimento explícito (`APP_ENV=development`), `X-Dev-User-ID` e `X-Dev-Tenant-ID` podem ser usados para operação local controlada. Esses headers são ignorados em produção.

## CORS

Em desenvolvimento local, `CORS_ALLOWED_ORIGINS=*` mantém a demo em `http://localhost:3000` funcional. Em produção, `CORS_ALLOWED_ORIGINS` deve conter a lista explícita de origens permitidas, separadas por vírgula.

Rotas globais:

| Rota | Política |
|---|---|
| `GET /health` | Pública |
| `GET /ready` | Pública; depende de PostgreSQL acessível |
| `GET /api/v1/tenants` | `PLATFORM_ADMIN` |
| `POST /api/v1/tenants` | `PLATFORM_ADMIN` |
| `POST /api/v1/admin/tenants` | `PLATFORM_ADMIN`; cria tenant e opcionalmente `TENANT_ADMIN` |
| `GET /api/v1/admin/dashboard` | `PLATFORM_ADMIN` |
| `GET /api/v1/admin/transactions` | `PLATFORM_ADMIN` |
| `GET /api/v1/admin/providers` | `PLATFORM_ADMIN` |
| `POST /api/v1/admin/providers` | `PLATFORM_ADMIN` |
| `GET /api/v1/admin/providers/:id` | `PLATFORM_ADMIN` |
| `PUT /api/v1/admin/providers/:id` | `PLATFORM_ADMIN` |
| `POST /api/v1/admin/providers/:id/activate` | `PLATFORM_ADMIN` |
| `POST /api/v1/admin/providers/:id/deactivate` | `PLATFORM_ADMIN` |
| `GET /api/v1/me/tenants` | JWT autenticado; retorna somente tenants das claims |
| `POST /api/v1/users` | JWT autenticado e tenant do body autorizado |
| `GET /api/v1/users/:id` | JWT autenticado e tenant do usuário autorizado |
| `/api/v1/providers/*` | JWT autenticado; quando `provider_id` é informado, `tenant_id` deve estar autorizado |

## Health Check

### GET /health

Valida se o processo HTTP está vivo.

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

### GET /ready

Valida se a aplicação está pronta para receber tráfego. Essa rota depende de conectividade com PostgreSQL via `PingContext`.

```bash
curl -s http://localhost:8080/ready
```

Resposta esperada quando o banco está acessível:

```json
{
  "data": {
    "status": "ready"
  }
}
```

Quando o banco não está acessível, retorna HTTP `503 Service Unavailable` com `error.code = "NOT_READY"`.

## Tenants

## Auth

### POST /api/v1/auth/login

Autentica usuário por e-mail e senha.

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@middleware.local","password":"ChangeMe123456!"}'
```

Senha incorreta ou usuário inexistente retornam HTTP `401` com `Credenciais inválidas.`.

### POST /api/v1/tenants

Cria um tenant/empresa. Requer role `PLATFORM_ADMIN`.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant A","document":"12.345.678/0001-90","address":"Rua Um, 123","district":"Centro","city":"São Paulo","postal_code":"12345-678","state":"SP","country_code":"55","area_code":"11","phone_number":"99999-8888","webhook_url":"https://cliente.example.com/webhooks/boletos"}'
```

### GET /api/v1/tenants

Lista tenants cadastrados. Requer role `PLATFORM_ADMIN`.

```bash
curl -s http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <platform-admin-token>"
```

### GET /api/v1/me/tenants

Lista somente os tenants presentes nas claims `tenant_id`/`tenant_ids` do JWT autenticado.

```bash
curl -s http://localhost:8080/api/v1/me/tenants \
  -H "Authorization: Bearer <token>"
```

### POST /api/v1/admin/tenants

Cria um tenant e, opcionalmente, um `TENANT_ADMIN` associado exclusivamente ao tenant criado. Requer `PLATFORM_ADMIN`.

```bash
curl -s -X POST http://localhost:8080/api/v1/admin/tenants \
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Cliente Demonstração",
    "admin":{
      "name":"Administrador Cliente",
      "email":"cliente@demo.local",
      "password":"Cliente123456!"
    }
  }'
```

### GET /api/v1/tenants/:id

Busca um tenant por ID.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>
```

### PUT /api/v1/tenants/:id

Atualiza os dados cadastrais usados como sacado nas emissões e a URL que receberá webhooks. Requer `TENANT_ADMIN`. Nome, CNPJ, endereço, bairro, cidade, CEP, UF, DDI, DDD e celular são obrigatórios; `webhook_url` é opcional e, quando informado, deve usar HTTP ou HTTPS.

## Dashboard

### Semântica financeira

`amount_cents`, volume financeiro, volume por tenant/provider/status e timeline financeira somam somente boletos efetivamente emitidos: `ISSUED`, `PAID`, `EXPIRED` e `CANCELLED`.

`CREATED`, `PROCESSING` e `FAILED` não entram em volume emitido.

Taxa de sucesso usa `(ISSUED + PAID + EXPIRED + CANCELLED) / (ISSUED + PAID + EXPIRED + CANCELLED + FAILED)`.

Ticket médio usa `volume_emitido / quantidade_emitida_com_sucesso`.

### GET /api/v1/admin/dashboard

Retorna indicadores globais da plataforma. Requer `PLATFORM_ADMIN`.

Filtros opcionais:

- `from`
- `to`
- `tenant_id`
- `provider_id`
- `status`
- `document`
- `email`
- `external_id`
- `our_number`

```bash
curl -s "http://localhost:8080/api/v1/admin/dashboard?from=2026-07-01&to=2026-07-31" \
  -H "Authorization: Bearer <platform-admin-token>"
```

Resposta resumida:

```json
{
  "data": {
    "totals": {
      "tenants": 10,
      "boletos": 1000,
      "amount_cents": 50000000,
      "issued": 900,
      "paid": 700,
      "failed": 10,
      "success_rate": 0.9,
      "failure_rate": 0.01,
      "average_ticket_cents": 50000
    },
    "by_tenant": [],
    "by_provider": [],
    "by_status": [],
    "timeline": []
  }
}
```

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
- `taxa_sucesso`
- `taxa_falha`
- `ticket_medio`
- `by_status`

### GET /api/v1/admin/transactions

Lista transações globais da plataforma com paginação. Requer `PLATFORM_ADMIN`.

Filtros opcionais:

- `from`
- `to`
- `tenant_id`
- `provider_id`
- `status`
- `document`
- `email`
- `external_id`
- `our_number`
- `limit`
- `offset`

```bash
curl -s "http://localhost:8080/api/v1/admin/transactions?status=ISSUED&limit=50&offset=0" \
  -H "Authorization: Bearer <platform-admin-token>"
```

### GET /api/v1/tenants/:tenantId/transactions

Lista transações do tenant autenticado com paginação. O `tenantId` vem da rota e é validado contra o JWT.

Filtros opcionais: `from`, `to`, `provider_id`, `status`, `document`, `email`, `external_id`, `our_number`, `limit` e `offset`.

```bash
curl -s "http://localhost:8080/api/v1/tenants/<tenantId>/transactions?email=cliente%40teste.com&limit=50&offset=0" \
  -H "Authorization: Bearer <tenant-token>"
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

### POST /api/v1/admin/providers

Cria um provider no catálogo global da plataforma. Requer `PLATFORM_ADMIN`.

```bash
curl -s -X POST http://localhost:8080/api/v1/admin/providers \
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Mock",
    "type":"BANK",
    "config":"{\"delay_ms\":0}"
  }'
```

Exemplo para Moncalieri Capital, sem credencial real:

```bash
curl -s -X POST http://localhost:8080/api/v1/admin/providers \
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Moncalieri Capital",
    "type":"BANK",
    "config":"{\"base_url\":\"https://dev.moncaliericapital.com.br\",\"api_key\":\"REPLACE_WITH_SECRET\",\"codigo_canal\":0,\"codigo_cliente\":0,\"timeout_seconds\":30,\"instrucoes\":\"Pagar ate o vencimento.\"}"
  }'
```

### GET /api/v1/admin/providers

Lista o catálogo global de providers. Requer `PLATFORM_ADMIN`.

```bash
curl -s http://localhost:8080/api/v1/admin/providers \
  -H "Authorization: Bearer <platform-admin-token>"
```

### GET /api/v1/admin/providers/:id

Busca um provider do catálogo global. `config` é mascarado quando existir.

### PUT /api/v1/admin/providers/:id

Atualiza nome, tipo, status, external ID, metadata e config global. Requer `PLATFORM_ADMIN`.

### POST /api/v1/admin/providers/:id/activate

Ativa um provider global.

### POST /api/v1/admin/providers/:id/deactivate

Desativa um provider global.

### GET /api/v1/tenants/:tenantId/providers

Lista providers habilitados para o tenant autenticado. Configurações sensíveis são mascaradas.

```bash
curl -s http://localhost:8080/api/v1/tenants/<tenantId>/providers
```

### GET /api/v1/tenants/:tenantId/providers/:id

Busca provider habilitado por ID dentro do tenant.

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

### Boleto Proposta

O boleto proposta permite iniciar uma cobrança sem cadastro prévio de customer. A entrada mínima do fluxo é:

- `email`
- `amount_cents`
- `due_date`
- `provider_id`

Fluxo:

```text
email + valor + vencimento
-> boleto CREATED
-> Compliance EMAIL
-> provider autorizado para o tenant
-> emissão
```

`customer_id` é opcional. Quando informado, a emissão segue o fluxo tradicional com dados completos do customer. Quando não informado, o boleto usa o email do destinatário e a resposta expõe esse valor em `recipient_email`.

### POST /api/v1/tenants/:tenantId/boletos

Cria uma intenção de boleto. Para boleto proposta, o handler aceita o campo JSON `email` e persiste o valor normalizado em `recipient_email`.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/boletos \
  -H "Authorization: Bearer <tenant-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "email":"cliente+demo@teste.com",
    "provider_id":"<providerId>",
    "amount_cents":15000,
    "due_date":"2026-07-30",
    "external_id":"pedido-123"
  }'
```

Campos principais:

| Campo | Obrigatório | Observação |
|---|---:|---|
| `email` | Sim, quando `customer_id` não for informado | Email do destinatário; normalizado com trim e lowercase. |
| `customer_id` | Não | UUID do cliente/sacado para o fluxo tradicional. |
| `provider_id` | Sim para emissão | UUID do provider habilitado para o tenant. |
| `amount_cents` | Sim | Valor em centavos. Deve ser maior que zero. |
| `due_date` | Sim | Data no formato `YYYY-MM-DD`. |
| `status` | Não | Se vazio, assume `CREATED`. |
| `external_id` | Não | Idempotência por `tenant_id + external_id` para registros ativos. |

Duplicidade de `external_id` no mesmo tenant retorna HTTP `409 Conflict` com `error.code = "DUPLICATE_RESOURCE"`. O mesmo `external_id` pode existir em tenants diferentes.

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
  -H "Authorization: Bearer <tenant-admin-token>" \
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
| `base64` | PDF do boleto em Base64, quando retornado pelo provider |
| `issued_at` | Timestamp da emissão simulada |

Para Mock, boletos proposta são emitidos com apenas email, valor e vencimento.

Na Moncalieri, o sacado é sempre preenchido com os dados completos do tenant. O e-mail continua sendo o destinatário informado na criação. A integração envia `RetornarBase64=true`, persiste o PDF devolvido e retorna seu conteúdo no campo `base64`.

Para Moncalieri, o adapter atual mapeia `Data.NossoNumero`, `Data.LinhaDigitavel` e `Data.CodigoBarras` da API do provider. A config aceita JSON com `base_url`, `api_key`, `codigo_canal`, `codigo_cliente`, `timeout_seconds` e `instrucoes`. O mapper Moncalieri ainda exige dados completos de sacado (`document`, `name`, endereço, bairro, cidade, CEP e UF`) no payload padronizado do provider; se faltar campo obrigatório, a API retorna `INVALID_REQUEST`.

## Compliance

### POST /api/v1/tenants/:tenantId/blacklist

Cria um bloqueio no tenant. Na Etapa 5, o fluxo principal é bloquear email de destinatário.

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants/<tenantId>/blacklist \
  -H "Authorization: Bearer <tenant-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "entry_type":"EMAIL",
    "value":"cliente+demo@teste.com",
    "reason":"Cliente solicitou não receber novas emissões",
    "source":"MANUAL"
  }'
```

Campos:

| Campo | Observação |
|---|---|
| `entry_type` | `EMAIL` ou `DOCUMENT`. Se omitido, a API mantém inferência legada por `document`/`value`. |
| `value` | Email ou documento a bloquear, conforme `entry_type`. |
| `document` | Compatibilidade para fluxo legado `DOCUMENT`; normalizado para somente números. |
| `name` | Nome de referência. |
| `reason` | Motivo exibido em consultas de bloqueio. |
| `notes` | Observações internas. |
| `source` | `MANUAL`, `API` ou `IMPORT`; assume `MANUAL` se vazio. |

Duplicidade ativa do mesmo `entry_type + value_normalized` no mesmo tenant retorna `DUPLICATE_RESOURCE`.

Emissão de boleto proposta para email bloqueado retorna:

```json
{
  "error": {
    "code": "RECIPIENT_BLOCKED",
    "message": "Este destinatário está bloqueado para novas emissões."
  }
}
```

### GET /api/v1/tenants/:tenantId/blacklist

Lista bloqueios do tenant. Aceita filtros `q` e `active`; use `q=<email>` para localizar bloqueios por destinatário.

```bash
curl -s "http://localhost:8080/api/v1/tenants/<tenantId>/blacklist?q=cliente%2Bdemo%40teste.com&active=true" \
  -H "Authorization: Bearer <tenant-admin-token>"
```

### GET /api/v1/tenants/:tenantId/blacklist/check?document=...

Consulta legada otimizada para saber se um documento está bloqueado. O código atual não possui endpoint dedicado de check por email; para email, use a listagem com `q` e `active=true`.

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
- Emissão de boleto proposta bloqueada automaticamente para emails ativos na blacklist do tenant.
- Emissão tradicional bloqueada automaticamente para documentos ativos na blacklist do tenant.

## Limitações conhecidas

- Não há endpoint dedicado para associar provider a tenant existente; essa associação acontece no onboarding `POST /api/v1/admin/tenants`.
- `GET /api/v1/tenants/:tenantId/blacklist/check` consulta apenas documentos.
- Migrations não executam automaticamente no startup; rode explicitamente o comando `migrate` antes de iniciar ou promover uma release.
