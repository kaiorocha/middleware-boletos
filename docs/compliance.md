# Compliance

## Objetivo

O módulo de Compliance impede que boletos sejam emitidos para CPF/CNPJ que solicitaram não receber novas cobranças. A regra é aplicada no backend, antes de qualquer chamada a provider bancário.

## Blacklist por tenant

Cada tenant mantém uma blacklist própria. Um documento bloqueado no Tenant A não bloqueia emissões no Tenant B.

Os documentos são normalizados para somente números antes de qualquer persistência ou consulta. A tabela `blacklist` possui índice único parcial para impedir duplicidade ativa de `tenant_id + document`.

## Fluxo de emissão

```text
API / Painel / Job
↓
BoletoService.Emit
↓
Valida dependências obrigatórias
↓
Busca boleto
↓
Busca customer
↓
Valida CPF/CNPJ do customer
↓
Compliance / Blacklist IsBlocked
↓
Bloqueado?
├─ SIM → CUSTOMER_BLOCKED
└─ NÃO → PayerBuilder
        ↓
        ProviderFactory
        ↓
        IssueBoleto
```

## Fail-closed

Compliance é dependência obrigatória do fluxo central de emissão. Se `BlacklistService`, repositories, provider factory ou payer builder não estiverem configurados, `BoletoService.Emit` interrompe a emissão.

Cliente sem documento também interrompe a emissão. A ausência de CPF/CNPJ nunca faz a API pular a validação de blacklist.

## CUSTOMER_BLOCKED

Quando o documento do customer está bloqueado, a API retorna:

```http
HTTP/1.1 409 Conflict
```

```json
{
  "error": {
    "code": "CUSTOMER_BLOCKED",
    "message": "Este cliente está bloqueado para novas emissões."
  }
}
```

Nesse cenário, `ProviderFactory` e `IssueBoleto` não são chamados.

## Garantia contra bypass

A chamada efetiva a `adapter.IssueBoleto` existe apenas dentro de `BoletoService.Emit`. Handlers, painel, APIs externas, jobs, filas e integrações futuras devem delegar emissão para esse fluxo central.

Testes de adapter podem chamar `IssueBoleto` diretamente apenas para validar a implementação isolada do provider.

## Auditoria

Eventos de compliance são registrados em `audit_logs`:

- `CustomerBlocked`
- `CustomerUnblocked`
- `BlockedEmissionAttempt`

Tentativas bloqueadas registram tenant, customer, documento, boleto, provider quando disponível, timestamp e reason. Dados sensíveis desnecessários não devem ser registrados.

## Operações

- `Block`: ativa ou reativa bloqueio.
- `Unblock`: desativa bloqueio mantendo o registro.
- `SoftDelete`: marca o registro como removido logicamente.
- `IsBlocked`: consulta otimizada usada pelo fluxo de emissão e pelo endpoint `check`.

## Isolamento multi-tenant

Todas as queries de blacklist exigem `tenant_id`. Nenhum endpoint deve listar, consultar, alterar ou excluir blacklist de outro tenant.

## Autorização usuário → tenant

Rotas tenant-scoped são protegidas por autorização centralizada:

- Sem identidade autenticada: HTTP `401` com `UNAUTHORIZED`.
- Identidade autenticada sem acesso ao tenant solicitado: HTTP `403` com `FORBIDDEN`.
- Identidade autorizada: a requisição segue para o handler.

Enquanto não há autenticação completa, a API usa uma camada explícita e testável baseada em headers:

- `X-User-ID`
- `X-Tenant-ID` ou `X-Tenant-IDs`

Em desenvolvimento, e somente com `APP_ENV=development`, o header `X-Dev-Tenant-ID` pode ser usado para operar localmente. Esse modo não é habilitado por padrão em produção.
