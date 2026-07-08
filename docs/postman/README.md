# Postman Collection

Esta pasta contém as collections Postman do projeto `middleware-boletos`.

## Como importar

1. Abra o Postman.
2. Clique em **Import**.
3. Selecione o arquivo `docs/postman/middleware-boletos-etapa-3.postman_collection.json`.
4. Confirme a importação da collection `middleware-boletos - Etapa 3`.

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

1. Create Tenant
2. Create Customer
3. Create Mock Provider
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

Os requests de criação capturam automaticamente os IDs retornados em `data.id` e salvam nas variáveis da collection:

- `tenantId`
- `customerId`
- `providerId`
- `boletoId`
- `moncalieriProviderId`
- `moncalieriBoletoId`
- `moncalieriApiKey` usa o placeholder `REPLACE_WITH_SECRET`; substitua apenas localmente.

Com isso, os requests seguintes conseguem reutilizar os IDs sem preenchimento manual.

## Collections disponíveis

- `middleware-boletos-etapa-3.postman_collection.json`: arquitetura de provedores, MockProvider, MoncalieriProvider, emissão simulada, health, balance e webhook.
- `middleware-boletos-etapa-2.postman_collection.json`: histórico da etapa anterior.

## Validações negativas

A collection da Etapa 3 inclui requests esperados com HTTP `400` e `409`:

- `Invalid Webhook - 400` valida `error.code = WEBHOOK_VALIDATION_ERROR`.
- `Duplicate Mock Provider - 409` valida `error.code = DUPLICATE_RESOURCE`.

A collection da Etapa 2 mantém as pastas históricas `Validation Errors` e `Duplicate Validation`.
