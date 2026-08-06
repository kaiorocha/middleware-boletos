# Postman Collection

Esta pasta contém as collections Postman do projeto `middleware-boletos`.

## Como importar

1. Abra o Postman.
2. Clique em **Import**.
3. Selecione o arquivo `docs/postman/middleware-boletos-etapa-4.postman_collection.json`.
4. Confirme a importação da collection `middleware-boletos - Etapa 4`.

## Como rodar o backend localmente

Na raiz do projeto, execute:

```bash
docker-compose up --build
```

O backend ficará disponível em:

```text
http://localhost:8080
```

## Ordem recomendada de execução

1. Login Platform Admin
2. Create Tenant With Tenant Admin - 201
3. Login Tenant Admin
4. Create Customer
5. Create Mock Provider
4. Create Moncalieri Provider
5. Create Boleto
6. Create Moncalieri Boleto
7. Emit Boleto
8. Emit Moncalieri Boleto
9. Provider Health
10. Moncalieri Provider Health
11. Provider Balance
12. Provider Webhook
13. Invalid Webhook - 400
14. Duplicate Mock Provider - 409
15. Dashboard Summary
16. Create Blacklist Entry
17. Check Blacklist Blocked
18. Create Blocked Boleto
19. Emit Blocked Boleto - 409
20. Create Boleto With External ID - 201
21. Duplicate External ID - 409
22. Create Boleto With Our Number - 201
23. Duplicate Our Number - 409
24. GET Tenants as Common User - 403
25. GET Tenants as Platform Admin - 200
26. POST Tenant as Common User - 403
27. POST Tenant as Platform Admin - 201
28. My Tenants - 200

Os requests de criação capturam automaticamente os IDs retornados em `data.id` e salvam nas variáveis da collection:

- `tenantId`
- `customerId`
- `providerId`
- `boletoId`
- `moncalieriProviderId`
- `moncalieriBoletoId`
- `moncalieriApiKey` usa o placeholder `REPLACE_WITH_SECRET`; substitua apenas localmente.

Com isso, os requests seguintes conseguem reutilizar os IDs sem preenchimento manual.

As rotas protegidas usam `Authorization: Bearer {{access_token}}`. Para execução local, a collection pode gerar um JWT HS256 automaticamente quando `autoGenerateAccessToken=true`, usando `jwtSecret`, `jwtIssuer`, `jwtAudience`, `jwtUserId` e o `tenantId` capturado.

Operações globais de tenant usam `{{platform_admin_access_token}}`, que inclui `roles: ["PLATFORM_ADMIN"]`.

Para validar contra um emissor real, preencha `access_token` manualmente e altere `autoGenerateAccessToken=false`.

## Collections disponíveis

- `middleware-boletos-etapa-4.postman_collection.json`: painel operacional, dashboard, blacklist e bloqueio de emissão por compliance.
- `middleware-boletos-etapa-3.postman_collection.json`: arquitetura de provedores, MockProvider, MoncalieriProvider, emissão simulada, health, balance e webhook.
- `middleware-boletos-etapa-2.postman_collection.json`: histórico da etapa anterior.

## Validações negativas

A collection da Etapa 4 inclui requests esperados com HTTP `400` e `409`:

- `Invalid Webhook - 400` valida `error.code = WEBHOOK_VALIDATION_ERROR`.
- `Duplicate Mock Provider - 409` valida `error.code = DUPLICATE_RESOURCE`.
- `Emit Blocked Boleto - 409` valida `error.code = CUSTOMER_BLOCKED`.
- A pasta `Duplicate Validation` valida duplicidade por tenant para users, customers, providers, `external_id` e `our_number`.
- A pasta `Auth Login` valida login do `PLATFORM_ADMIN`, login do `TENANT_ADMIN` e credenciais inválidas.
- A pasta `Auth Examples` valida `401 Unauthorized`, token malformado e `403 Forbidden` por cross-tenant.
- A pasta `RBAC` valida `PLATFORM_ADMIN` em `GET/POST /api/v1/tenants` e `GET /api/v1/me/tenants`.

A collection da Etapa 2 mantém a pasta histórica `Validation Errors`.
