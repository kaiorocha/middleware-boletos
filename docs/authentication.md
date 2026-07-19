# Authentication

## Visão geral

A API usa autenticação JWT por Bearer token para rotas administrativas e operacionais. A única rota pública é:

- `GET /health`

Todas as demais rotas exigem:

```http
Authorization: Bearer <JWT>
```

## Estratégia atual

A implementação inicial usa JWT assinado com `HS256` e segredo compartilhado via `JWT_SECRET`. A validação está centralizada no pacote `backend/internal/auth`, mantendo a API preparada para trocar o validador por JWKS/OIDC no futuro.

O backend valida:

- assinatura HMAC-SHA256;
- algoritmo exatamente `HS256`;
- expiração (`exp`);
- issuer (`iss`), quando `JWT_ISSUER` estiver configurado;
- audience (`aud`), quando `JWT_AUDIENCE` estiver configurado;
- `sub` como UUID;
- `tenant_id` e/ou `tenant_ids` como UUIDs.

Tokens com `alg=none`, algoritmo diferente, assinatura inválida, expiração vencida, issuer inválido ou audience inválida retornam HTTP `401`.

## Claims esperadas

Token com tenant único:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655449999",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "exp": 1784500000,
  "iss": "middleware-boletos-local",
  "aud": "middleware-boletos-api"
}
```

Token com múltiplos tenants:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655449999",
  "tenant_ids": [
    "550e8400-e29b-41d4-a716-446655440000",
    "550e8400-e29b-41d4-a716-446655440099"
  ],
  "exp": 1784500000,
  "iss": "middleware-boletos-local",
  "aud": "middleware-boletos-api"
}
```

## Autorização por tenant

Após validar o JWT, a API cria uma identidade autenticada no contexto da request. Rotas tenant-scoped com `/api/v1/tenants/{tenantId}/...` só continuam quando `{tenantId}` está presente em `tenant_id` ou `tenant_ids`.

- Sem token ou token inválido: HTTP `401` com `UNAUTHORIZED`.
- Token válido sem acesso ao tenant da URL: HTTP `403` com `FORBIDDEN`.

Headers arbitrários como `X-User-ID`, `X-Tenant-ID` e `X-Tenant-IDs` não autenticam usuários em produção.

## Modo development

Com `APP_ENV=development`, o backend aceita um atalho explícito para desenvolvimento local:

```http
X-Dev-User-ID: <uuid>
X-Dev-Tenant-ID: <uuid>
```

Esse modo não é habilitado por fallback. Se `APP_ENV` estiver ausente, a API assume `production` e ignora os headers de desenvolvimento.

## Configuração

Variáveis do backend:

```env
APP_ENV=production
JWT_SECRET=<segredo-forte>
JWT_ISSUER=<issuer-esperado>
JWT_AUDIENCE=<audience-esperada>
```

Em produção, `JWT_SECRET` é obrigatório. Se estiver ausente, a API falha no startup para evitar modo inseguro.

