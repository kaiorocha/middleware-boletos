# Roles and Permissions

## Contextos

### PLATFORM_ADMIN

Gerencia a plataforma como um todo:

- dashboard global;
- transações de todos os tenants;
- onboarding de tenants;
- catálogo global de providers;
- associação de providers aos tenants;
- administradores globais via bootstrap seguro.

### TENANT_ADMIN

Opera somente os tenants presentes no JWT:

- dashboard do próprio tenant;
- consulta de transações, boletos, clientes e providers habilitados;
- gestão de compliance/blacklist do próprio tenant;
- consulta e gestão de usuários do próprio tenant.

### TENANT_USER

Consulta somente dados do próprio tenant:

- dashboard;
- transações;
- boletos;
- clientes;
- providers habilitados;
- compliance em modo leitura.

## Matriz resumida

| Recurso | PLATFORM_ADMIN | TENANT_ADMIN | TENANT_USER | API tenant-scoped |
|---|---:|---:|---:|---:|
| Dashboard global | Sim | Não | Não | Não |
| Transações globais | Sim | Não | Não | Não |
| Criar tenant | Sim | Não | Não | Não |
| Catálogo de providers | Sim | Não | Não | Não |
| Associar provider ao tenant | Sim | Não | Não | Não |
| Dashboard do tenant | Não | Sim | Sim | Sim |
| Consultar boletos/clientes/providers do tenant | Não | Sim | Sim | Sim |
| Criar clientes/boletos | Não pelo painel | Não pelo painel | Não | Sim |
| Emitir boletos | Não pelo painel | Não pelo painel | Não | Sim |
| Compliance do tenant | Não | Sim | Leitura | Sim |
| Usuários do tenant | Não | Sim | Não | Parcial |

O painel do tenant é operacional e consultivo. A criação de clientes, boletos e solicitações de emissão permanece no contrato REST tenant-scoped para integrações externas.
