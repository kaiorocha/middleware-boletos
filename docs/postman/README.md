# Postman Collection

Esta pasta contém a collection Postman da Etapa 2 do projeto `middleware-boletos`.

## Como importar

1. Abra o Postman.
2. Clique em **Import**.
3. Selecione o arquivo `docs/postman/middleware-boletos-etapa-2.postman_collection.json`.
4. Confirme a importação da collection `middleware-boletos - Etapa 2`.

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

1. Health
2. Create Tenant
3. Create User
4. Create Customer
5. Create Provider
6. Create Boleto
7. List/Get resources

Os requests de criação capturam automaticamente os IDs retornados em `data.id` e salvam nas variáveis da collection:

- `tenantId`
- `userId`
- `customerId`
- `providerId`
- `boletoId`

Com isso, os requests seguintes conseguem reutilizar os IDs sem preenchimento manual.

## Validações negativas

A pasta `Validation Errors` contém requests esperados com HTTP `400`, cobrindo payloads inválidos e validando a presença do campo `error` na resposta.

A pasta `Duplicate Validation` contém requests esperados com HTTP `409`, cobrindo duplicidades por tenant e validando `error.code = DUPLICATE_RESOURCE`.
