# Provider Management

## Modelo

Providers são gerenciados pela plataforma em duas camadas:

- `providers`: catálogo global de integrações disponíveis.
- `tenant_providers`: associação entre tenant e provider, com flag `active` e configuração opcional por vínculo.

Um tenant só pode usar um provider quando:

- o provider existe no catálogo;
- o provider global está `ACTIVE`;
- existe associação em `tenant_providers`;
- a associação está ativa e sem soft delete.

Caso contrário, a emissão retorna `PROVIDER_NOT_ALLOWED`.

## Endpoints

### Catálogo global

Requer `PLATFORM_ADMIN`.

- `GET /api/v1/admin/providers`
- `POST /api/v1/admin/providers`

### Providers habilitados do tenant

Requer JWT com acesso ao tenant.

- `GET /api/v1/tenants/:tenantId/providers`
- `GET /api/v1/tenants/:tenantId/providers/:providerId`

O tenant não cria providers pelo painel nem pelo endpoint tenant-scoped. Providers são criados e habilitados pela plataforma.

## Onboarding

`POST /api/v1/admin/tenants` cria:

- tenant;
- administrador inicial opcional;
- associações iniciais de providers quando informadas.

Se a criação do admin ou qualquer associação de provider falhar, o tenant recém-criado é removido para evitar onboarding parcial.

Exemplo:

```json
{
  "name": "Cliente Demonstração",
  "admin": {
    "name": "Administrador Cliente",
    "email": "cliente@demo.local",
    "password": "Cliente123456!"
  },
  "providers": [
    {
      "provider_id": "550e8400-e29b-41d4-a716-446655440002",
      "active": true
    }
  ]
}
```
