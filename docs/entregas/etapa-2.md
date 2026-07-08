# Entrega — Etapa 2: Backend e API

## Projeto

Middleware para emissão e gestão de boletos bancários.

## Objetivo da etapa

A **Etapa 2 — Backend e API** teve como objetivo transformar a fundação técnica criada na Etapa 1 em uma API backend funcional, com persistência em PostgreSQL, estrutura modular, repositories, services, validações, rotas REST e testes básicos.

Nesta etapa, o boleto ainda não é emitido em banco ou provedor real. A API registra a intenção de boleto, preparando a base para a integração bancária da Etapa 3.

## Status

**Concluída e pronta para validação/aceite.**

## Entregáveis concluídos

### 1. Configuração da aplicação

- Pacote `internal/config` criado.
- Leitura de variáveis de ambiente.
- Configuração de:
  - `PORT`
  - `DATABASE_URL`
  - `REDIS_URL`
  - `BACKEND_ENV`
- Fallbacks para ambiente local.

### 2. Banco de dados e migrations

- Conexão com PostgreSQL implementada.
- Execução automática de migrations na inicialização da aplicação.
- Controle de migrations executadas via tabela `schema_migrations`.
- Migrations SQL versionadas adicionadas para:
  - `tenants`
  - `users`
  - `customers`
  - `providers`
  - `boletos`
  - `webhook_events`
  - `audit_logs`
  - campos auxiliares de status e referência externa.

### 3. Domínio

Entidades atualizadas em `internal/domain`:

- `Tenant`
- `User`
- `Customer`
- `Provider`
- `Boleto`
- `WebhookEvent`
- `AuditLog`

Campos adicionados quando aplicável:

- `status`
- `external_id`
- `deleted_at`
- timestamps de criação e atualização.

### 4. Repositories

Camada de acesso a dados criada em `internal/repository` para:

- `TenantRepo`
- `UserRepo`
- `CustomerRepo`
- `ProviderRepo`
- `BoletoRepo`
- `WebhookEventRepo`
- `AuditLogRepo`

Funcionalidades contempladas conforme aplicável:

- criação;
- busca por ID;
- listagem por tenant;
- atualização;
- exclusão lógica quando aplicável.

### 5. Services

Camada de regras de negócio criada em `internal/service` para:

- `TenantService`
- `UserService`
- `CustomerService`
- `ProviderService`
- `BoletoService`

Responsabilidades implementadas:

- validação de campos obrigatórios;
- validação de UUID;
- validação de e-mail;
- validação de valor e vencimento de boleto;
- status inicial de boleto como `CREATED` ou `PENDING`;
- preparação para isolamento por tenant.

### 6. API REST

Rotas REST implementadas:

#### Health

- `GET /health`

#### Tenants

- `POST /api/v1/tenants`
- `GET /api/v1/tenants`
- `GET /api/v1/tenants/:id`

#### Users

- `POST /api/v1/users`
- `GET /api/v1/users/:id`
- `GET /api/v1/tenants/:tenantId/users`

#### Customers

- `POST /api/v1/tenants/:tenantId/customers`
- `GET /api/v1/tenants/:tenantId/customers`
- `GET /api/v1/tenants/:tenantId/customers/:id`
- `PUT /api/v1/tenants/:tenantId/customers/:id`

#### Providers

- `POST /api/v1/tenants/:tenantId/providers`
- `GET /api/v1/tenants/:tenantId/providers`
- `GET /api/v1/tenants/:tenantId/providers/:id`

#### Boletos

- `POST /api/v1/tenants/:tenantId/boletos`
- `GET /api/v1/tenants/:tenantId/boletos`
- `GET /api/v1/tenants/:tenantId/boletos/:id`

### 7. Padrão de resposta

#### Sucesso

```json
{
  "data": {}
}
```

#### Erro

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Descrição do erro"
  }
}
```

### 8. Testes

Testes básicos adicionados para:

- endpoint `GET /health`;
- validações de `TenantService`;
- validações de `UserService`;
- validações de `CustomerService`;
- validações de `ProviderService`;
- validações de `BoletoService`.

Casos de boleto cobertos:

- rejeição de `tenant_id` inválido;
- rejeição de `customer_id` inválido;
- rejeição de `provider_id` inválido quando informado;
- rejeição de valor menor ou igual a zero;
- rejeição de vencimento vazio;
- rejeição de status inválido;
- criação válida com status `CREATED`;
- criação válida com status `PENDING`;
- criação válida com provider opcional.

### 9. CI/CD

Workflow GitHub Actions mantido e ajustado para:

- branches `develop`, `main` e `feature/*`;
- execução de `go vet ./...`;
- execução de `go test ./...`;
- build do frontend.

### 10. Documentação

Documentações atualizadas/adicionadas:

- `README.md` com status da Etapa 2, rotas e exemplos;
- `docs/api.md` com documentação detalhada da API;
- `backend/internal/storage/migrations/README.md` com funcionamento das migrations versionadas;
- `docs/entregas/etapa-2.md` com resumo formal da entrega.

## Como validar localmente

### Subir ambiente

```bash
cp .env.example .env
docker-compose up --build
```

### Health check

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

### Rodar testes

```bash
cd backend
go test ./...
```

## Critérios de aceite atendidos

- Backend compila.
- API inicia sem duplicidade de rotas.
- PostgreSQL conectado.
- Migrations versionadas aplicadas automaticamente.
- Entidades principais persistidas.
- Rotas REST base disponíveis.
- Boleto registrado como intenção, sem emissão bancária real.
- Validações básicas implementadas.
- Testes básicos adicionados.
- CI preparado para validação.
- Documentação atualizada.

## Limitações conhecidas

- Autenticação/autorização ainda não implementada.
- Integração bancária real ainda não implementada.
- Webhooks reais de provedores ainda não implementados.
- Sem rollback automático de migrations.
- Migrações de produção podem ser evoluídas futuramente com ferramenta dedicada.

## Próxima etapa

### Etapa 3 — Integração Bancária

Próximos objetivos:

- criar camada de adaptadores de provedores;
- implementar contrato comum de emissão de boleto;
- integrar primeiro provedor bancário/gateway;
- implementar consulta de status;
- preparar recebimento de webhooks;
- evoluir tratamento de erros e retries.
