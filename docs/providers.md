# Providers

## Visão geral

A integração bancária fica isolada em `backend/internal/providers`. O restante do sistema usa interfaces e tipos padronizados, sem depender de contratos específicos de bancos.

O catálogo de integrações é responsabilidade do `PLATFORM_ADMIN`. Tenants apenas consultam os providers habilitados para sua conta.

## Estrutura

- `contracts`: interface `ProviderAdapter` e interface de factory.
- `types`: requests/responses padronizados, status, health, balance e webhook event.
- `factory`: seleção centralizada de adapter por `provider.name`.
- `mock`: adapter usado em desenvolvimento e nesta etapa.
- `moncalieri`: adapter real para Moncalieri Capital.
- `base`: máquina de estados de boleto.
- `base/payer_builder.go`: conversão reutilizável de `domain.Customer` para `types.Payer`.
- `events`: eventos internos estruturados.
- `webhooks`: infraestrutura para receber, validar e converter webhooks.
- `validators`: validações compartilhadas de requests/events.
- `errors`: erro padronizado de provider.

## Fluxo de emissão

1. O boleto é criado com status `CREATED` e um `provider_id`.
2. A API chama `POST /api/v1/tenants/:tenantId/boletos/:id/emit`.
3. `BoletoService` valida tenant, boleto, provider e estado.
4. `BoletoService` busca o `Customer`.
5. `BoletoService` consulta a blacklist do tenant.
6. `BoletoService` confirma que o provider está ativo no catálogo e habilitado para o tenant em `tenant_providers`.
7. `BoletoService` monta `ProviderConfig` usando `tenant_providers.config`.
8. `DefaultPayerBuilder` converte `Customer` para `types.Payer`.
9. A factory recebe `provider.name` e retorna o adapter.
10. O adapter executa `IssueBoleto`.
11. O service persiste `status`, `external_id`, `barcode`, `digitable_line`, `our_number` e `issued_at`.

Provider não habilitado retorna HTTP `403` com `error.code = "PROVIDER_NOT_ALLOWED"`.

## PayerBuilder

Toda regra de pagador fica centralizada em `base.PayerBuilder`.

`DefaultPayerBuilder` valida e normaliza:

- `Name`
- `Document`
- `Email`
- `Address`
- `District`
- `City`
- `State`
- `PostalCode`

Normalizações aplicadas:

- documento sem máscara;
- CEP sem máscara;
- UF em uppercase;
- email com trim e lowercase;
- endereço, bairro e cidade com trim;
- endereço completo montado com endereço, número e complemento.

Se faltar dado obrigatório, retorna erro estruturado `INVALID_PAYER`. Novos providers não devem implementar conversões próprias de `Customer`; eles devem receber apenas `types.IssueRequest`.

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

## Provider Moncalieri Capital

O primeiro provider real implementado é `MoncalieriProvider`, em `backend/internal/providers/moncalieri`.

Nomes aceitos pela factory:

- `moncalieri`
- `moncaliericapital`
- `moncalieri-capital`
- `Moncalieri Capital`

### Configuração

O campo `Provider.Config` deve ser uma string JSON:

```json
{
  "base_url": "https://dev.moncaliericapital.com.br",
  "api_key": "REPLACE_WITH_SECRET",
  "codigo_canal": 0,
  "codigo_cliente": 0,
  "timeout_seconds": 30,
  "instrucoes": "Pagar ate o vencimento."
}
```

Nunca commitar `api_key` real. Use variáveis de ambiente, secret manager ou configuração segura na carga de dados do provider.

### Endpoints Usados

- `POST /api/CashIn/GerarBoleto`: emissão de boleto.
- `POST /api/CashIn/ConsultarBoleto`: consulta de boleto por `NossoNumero`.
- `POST /api/CashIn/ConsultarBoletoLote`: listagem por período.
- `POST /api/CashIn/SolicitarBaixaBoleto`: baixa/cancelamento.

### Operações Suportadas

- `IssueBoleto`
- `GetBoleto`
- `ListBoletos`
- `CancelBoleto`
- `Health`

`Health` valida a configuração local. A especificação OpenAPI enviada não possui endpoint dedicado de health.

### Operações Não Suportadas

- `GetBalance`: retorna `UNSUPPORTED_OPERATION`, pois a API enviada não possui endpoint de saldo.
- `RegisterWebhook`: retorna `UNSUPPORTED_OPERATION`.
- `ValidateWebhook`: retorna `UNSUPPORTED_OPERATION`.

### Dados obrigatórios do pagador

`IssueBoleto` recebe dados do pagador em `types.IssueRequest.Payer`. O `DefaultPayerBuilder` monta esse objeto a partir de `domain.Customer`.

Campos usados:

- `Document`
- `Name`
- `Email`
- `Address`
- `District`
- `City`
- `PostalCode`
- `State`

Campos obrigatórios para emissão real: `Name`, `Document`, `Address`, `District`, `City`, `State` e `PostalCode`. `Email` é normalizado quando informado, mas não bloqueia emissão se ausente. Se o customer não tiver endereço completo, o fluxo retorna `INVALID_PAYER` antes de chamar o provider.

### Mapeamento de Status

- `Pago`, `Pagos`, `Liquidado` -> `PAID`
- `Pendente`, `Pendentes`, `Registrado` -> `ISSUED`
- `Baixado`, `Baixados`, `Cancelado` -> `CANCELLED`
- `Vencido` -> `EXPIRED`
- status desconhecido -> `PROCESSING`

## Webhooks

`providers/webhooks.Receive` recebe o adapter e uma `ValidateWebhookRequest`. O adapter valida o payload e devolve um `types.WebhookEvent`. O `MockProvider` desserializa JSON padronizado e preenche `provider_id`, `tenant_id`, `id` e `received_at` quando necessário.

A documentação OpenAPI da Moncalieri enviada não descreve webhooks. Por isso, o adapter Moncalieri retorna `UNSUPPORTED_OPERATION` para registro e validação de webhook até que exista contrato oficial.

## Como adicionar um novo banco

1. Criar um pacote em `backend/internal/providers/<banco>`.
2. Implementar todos os métodos de `contracts.ProviderAdapter`.
3. Converter requests/responses específicos do banco para os tipos de `providers/types`.
4. Adicionar o banco no `switch` central de `factory.ProviderFactory`.
5. Cobrir issue, health, balance, webhook validation e erros com testes.

Nenhuma chamada específica do banco deve ser adicionada nos services ou handlers.
