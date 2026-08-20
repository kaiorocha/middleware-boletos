# OpenAPI

Esta pasta contém a especificação OpenAPI publicável do `middleware-boletos`.

Arquivo principal:

```text
docs/openapi/middleware-boletos-etapa-5.openapi.json
```

## Como usar

O arquivo está em OpenAPI `3.0.3` e pode ser importado em ferramentas como Swagger UI, Redoc, Stoplight, Insomnia ou Postman.

Para publicar uma documentação estática, use o arquivo JSON como entrada da ferramenta escolhida. Exemplo com Swagger UI local:

```bash
docker run --rm -p 8081:8080 \
  -e SWAGGER_JSON=/openapi/middleware-boletos-etapa-5.openapi.json \
  -v "$PWD/docs/openapi:/openapi" \
  swaggerapi/swagger-ui
```

Depois acesse:

```text
http://localhost:8081
```

## Escopo

A especificação cobre os endpoints principais para integração de estabelecimentos:

- autenticação;
- consulta e atualização cadastral do tenant;
- providers habilitados para o tenant;
- criação e emissão de boleto proposta;
- consulta de boletos;
- transações paginadas;
- Compliance por email/documento;
- health/readiness;
- endpoints administrativos necessários para onboarding e catálogo de providers.

Os schemas refletem a resposta padrão da API:

- sucesso em `{ "data": ... }`;
- erro em `{ "error": { "code": "...", "message": "..." } }`.
