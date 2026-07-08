# Providers

## Visão geral

A integração bancária fica isolada em `backend/internal/providers`. O restante do sistema usa interfaces e tipos padronizados, sem depender de contratos específicos de bancos.

## Estrutura

- `contracts`: interface `ProviderAdapter` e interface de factory.
- `types`: requests/responses padronizados, status, health, balance e webhook event.
- `factory`: seleção centralizada de adapter por `provider.name`.
- `mock`: adapter usado em desenvolvimento e nesta etapa.
- `base`: máquina de estados de boleto.
- `events`: eventos internos estruturados.
- `webhooks`: infraestrutura para receber, validar e converter webhooks.
- `validators`: validações compartilhadas de requests/events.
- `errors`: erro padronizado de provider.

## Fluxo de emissão

1. O boleto é criado com status `CREATED` e um `provider_id`.
2. A API chama `POST /api/v1/tenants/:tenantId/boletos/:id/emit`.
3. `BoletoService` valida tenant, boleto, provider e estado.
4. A factory recebe `provider.name` e retorna o adapter.
5. O adapter executa `IssueBoleto`.
6. O service persiste `status`, `external_id`, `barcode`, `digitable_line`, `our_number` e `issued_at`.

## Estados

Status suportados:

- `CREATED`
- `PROCESSING`
- `ISSUED`
- `FAILED`
- `CANCELLED`
- `PARTIAL`
- `PAID`
- `EXPIRED`

Transições principais:

- `CREATED -> PROCESSING`
- `PROCESSING -> ISSUED`
- `PROCESSING -> FAILED`
- `PROCESSING -> CANCELLED`
- `ISSUED -> PAID`
- `ISSUED -> PARTIAL`
- `ISSUED -> EXPIRED`
- `ISSUED -> CANCELLED`
- `PARTIAL -> PAID`

## Webhooks

`providers/webhooks.Receive` recebe o adapter e uma `ValidateWebhookRequest`. O adapter valida o payload e devolve um `types.WebhookEvent`. Nesta etapa, o `MockProvider` desserializa JSON padronizado e preenche `provider_id`, `tenant_id`, `id` e `received_at` quando necessário.

## Como adicionar um novo banco

1. Criar um pacote em `backend/internal/providers/<banco>`.
2. Implementar todos os métodos de `contracts.ProviderAdapter`.
3. Converter requests/responses específicos do banco para os tipos de `providers/types`.
4. Adicionar o banco no `switch` central de `factory.ProviderFactory`.
5. Cobrir issue, health, balance, webhook validation e erros com testes.

Nenhuma chamada específica do banco deve ser adicionada nos services ou handlers.
