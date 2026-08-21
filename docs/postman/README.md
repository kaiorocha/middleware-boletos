# Postman Collection

Esta pasta contém as collections Postman do projeto `middleware-boletos`.

## Collection Recomendada

Use a collection da Etapa 5:

```text
docs/postman/middleware-boletos-etapa-5.postman_collection.json
```

Ela valida o fluxo atual de boleto proposta, Compliance por email, Production Readiness e integração com providers.

As collections das etapas 2, 3 e 4 permanecem nesta pasta como histórico.

## Como importar

1. Abra o Postman.
2. Clique em **Import**.
3. Selecione `docs/postman/middleware-boletos-etapa-5.postman_collection.json`.
4. Confirme a importação da collection `middleware-boletos - Etapa 5 Final`.

## Como rodar o backend localmente

Na raiz do projeto:

```bash
cp .env.example .env
docker-compose up -d postgres redis
docker-compose run --rm backend migrate
docker-compose up --build
```

O backend fica disponível em:

```text
http://localhost:8080
```

## Variáveis

A collection já define defaults seguros para execução local:

- `baseUrl = http://localhost:8080`
- `platformAdminEmail = admin@middleware.local`
- `platformAdminPassword = ChangeMe123456!`
- `tenantAdminPassword = Tenant123456!`
- `amountCents = 10000`

Também há defaults para os dados cadastrais obrigatórios do tenant:

- `tenantDocument`
- `tenantAddress`
- `tenantDistrict`
- `tenantCity`
- `tenantPostalCode`
- `tenantState`
- `tenantCountryCode`
- `tenantAreaCode`
- `tenantPhoneNumber`
- `tenantWebhookUrl`

E defaults equivalentes para o teste de atualização do tenant:

- `updatedTenantDocument`
- `updatedTenantAddress`
- `updatedTenantDistrict`
- `updatedTenantCity`
- `updatedTenantPostalCode`
- `updatedTenantState`
- `updatedTenantCountryCode`
- `updatedTenantAreaCode`
- `updatedTenantPhoneNumber`
- `updatedTenantWebhookUrl`

O pre-request script da collection gera automaticamente por execução:

- `tenantAdminEmail`
- `recipientEmail`
- `externalId`
- `idempotencyExternalId`
- `dueDate` com hoje + 7 dias

Os scripts de teste capturam automaticamente:

- `platform_admin_access_token`
- `tenant_admin_access_token`
- `tenantId`
- `mockProviderId`
- `boletoId`
- `blockedBoletoId`
- `blacklistEntryId`
- `moncalieriProviderId`
- `moncalieriTenantId`
- `moncalieri_tenant_admin_access_token`
- `moncalieriBoletoId`

## Ordem recomendada

1. Health
2. Ready
3. Login Platform Admin
4. Create Mock Provider
5. Create Tenant With Tenant Admin and Mock Provider
6. Login Tenant Admin
7. Get Tenant
8. Update Tenant
9. Create Proposal Boleto
10. Emit Proposal Boleto
11. Get Boleto
12. Tenant Transactions
13. Admin Transactions
14. Block Recipient Email
15. Create Blocked Proposal Boleto
16. Emit Blocked Proposal - RECIPIENT_BLOCKED
17. Unblock Email
18. Emit After Unblock
19. Idempotency External ID
20. Moncalieri Homologação

## Moncalieri Homologação

A pasta `06 - Moncalieri - Homologação` fica desabilitada por padrão para não executar sem credenciais.

Preencha apenas localmente:

- `moncalieriBaseUrl`
- `moncalieriApiKey`
- `moncalieriCodigoCanal`
- `moncalieriCodigoCliente`

O contrato implementado em código usa config JSON com `base_url`, `api_key`, `codigo_canal`, `codigo_cliente`, `timeout_seconds` e `instrucoes`. Não há suporte atual a `client_id` ou `client_secret`.

Não existe endpoint dedicado para associar provider a tenant existente. Para habilitar Moncalieri em um tenant, use o onboarding `POST /api/v1/admin/tenants` com o campo `providers`.

Como os requests dessa pasta ficam desabilitados, habilite e execute manualmente na ordem em que aparecem quando as credenciais de homologação estiverem configuradas.

## Collections disponíveis

- `middleware-boletos-etapa-5.postman_collection.json`: versão final da Etapa 5 com dados cadastrais do tenant, atualização de tenant, boleto proposta, Compliance por email, `RECIPIENT_BLOCKED`, idempotência e readiness.
- `middleware-boletos-etapa-4.postman_collection.json`: painel operacional, dashboard, blacklist e RBAC da etapa anterior.
- `middleware-boletos-etapa-3.postman_collection.json`: arquitetura de providers, MockProvider, MoncalieriProvider, health, balance e webhook.
- `middleware-boletos-etapa-2.postman_collection.json`: rotas CRUD iniciais e validações básicas.
