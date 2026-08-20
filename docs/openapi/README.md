# OpenAPI

Esta pasta contém dois contratos OpenAPI do `middleware-boletos`, com finalidades distintas.

Contratos:

```text
middleware-boletos-etapa-5.openapi.json   # interno/completo; não publicar
middleware-boletos-public.openapi.json    # integração externa; fonte do portal
```

## Como usar

Os arquivos estão em OpenAPI `3.0.3`. Para validar o contrato público:

```bash
bash scripts/docs/validate-public-openapi.sh
```

O portal Scalar e suas instruções locais ficam em `developer-portal/README.md`. Durante o build, `PUBLIC_API_PRODUCTION_URL` e o `PUBLIC_API_HML_URL` opcional substituem os servers somente em `dist/openapi.json`; o arquivo-fonte permanece imutável.

## Escopo

A especificação pública cobre somente os endpoints para integração de estabelecimentos:

- autenticação;
- descoberta dos tenants autorizados;
- providers habilitados para o tenant;
- criação e emissão de boleto proposta;
- consulta de boletos;
- transações paginadas;
- Compliance por email/documento;
- webhook de provider.

Health/readiness, onboarding global, catálogo global de providers e todo `/api/v1/admin/*` permanecem exclusivamente no contrato interno.

Os schemas refletem a resposta padrão da API:

- sucesso em `{ "data": ... }`;
- erro em `{ "error": { "code": "...", "message": "..." } }`.
