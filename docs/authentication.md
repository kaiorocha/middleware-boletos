# Authentication

## Visão geral

A API usa autenticação JWT por Bearer token para rotas administrativas e operacionais. A única rota pública é:

- `GET /health`
- `POST /api/v1/auth/login`

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
- issuer (`iss`);
- audience (`aud`);
- `sub` como UUID;
- `tenant_id` e/ou `tenant_ids` como UUIDs.
- `roles` como lista de strings normalizadas para uppercase.

Tokens com `alg=none`, algoritmo diferente, assinatura inválida, expiração vencida, issuer inválido ou audience inválida retornam HTTP `401`.

## Claims esperadas

Token com tenant único:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655449999",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "roles": [],
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
  "roles": [],
  "exp": 1784500000,
  "iss": "middleware-boletos-local",
  "aud": "middleware-boletos-api"
}
```

Token com administração global:

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655449999",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "roles": ["PLATFORM_ADMIN"],
  "exp": 1784500000,
  "iss": "middleware-boletos-local",
  "aud": "middleware-boletos-api"
}
```

## Login

```http
POST /api/v1/auth/login
```

```json
{
  "email": "admin@middleware.local",
  "password": "ChangeMe123456!"
}
```

Resposta:

```json
{
  "data": {
    "access_token": "...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
      "id": "...",
      "name": "Administrador",
      "email": "admin@middleware.local",
      "roles": ["PLATFORM_ADMIN"],
      "tenant_ids": []
    }
  }
}
```

Login inválido retorna HTTP `401` com a mensagem genérica `Credenciais inválidas.`.

## RBAC

Roles vêm exclusivamente do JWT validado. A API normaliza roles com trim + uppercase e remove duplicidades.

Role global atual:

- `PLATFORM_ADMIN`: permite administração global da plataforma.
- `TENANT_ADMIN`: administra apenas tenants presentes nas claims.
- `TENANT_USER`: role operacional inicial para leitura/uso futuro.

Rotas globais protegidas por `PLATFORM_ADMIN`:

- `GET /api/v1/tenants`
- `POST /api/v1/tenants`
- `POST /api/v1/admin/tenants`

Usuários sem `PLATFORM_ADMIN` recebem HTTP `403` nessas rotas, mesmo com JWT válido.

## Tenants do usuário

Usuários comuns não devem usar `GET /api/v1/tenants`. Para descobrir tenants acessíveis, use:

```http
GET /api/v1/me/tenants
```

Esse endpoint lê `tenant_id`/`tenant_ids` da Identity autenticada e retorna somente tenants presentes nas claims.

## Bootstrap

O primeiro `PLATFORM_ADMIN` pode ser criado por variáveis de ambiente:

```env
BOOTSTRAP_ADMIN_EMAIL=admin@middleware.local
BOOTSTRAP_ADMIN_PASSWORD=ChangeMe123456!
BOOTSTRAP_ADMIN_NAME=Administrador
```

O bootstrap roda somente quando as variáveis estão preenchidas e ainda não existe `PLATFORM_ADMIN`. Ele não deve ser usado com credenciais demo em produção.

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

Em produção, `JWT_SECRET`, `JWT_ISSUER` e `JWT_AUDIENCE` são obrigatórios. `JWT_SECRET` deve ter pelo menos 32 caracteres. Se qualquer requisito estiver ausente, a API falha no startup com `auth config invalid`.
