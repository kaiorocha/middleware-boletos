# Campanhas e emissão em lote

Campanhas agrupam boletos de um tenant sem alterar o comportamento de boletos antigos. O vínculo `campaign_id` é opcional.

## Permissões e rotas

Todas as rotas abaixo exigem autenticação e são isoladas pelo tenant. `TENANT_USER` mantém acesso de leitura; criação, alteração, importação e emissão exigem `TENANT_ADMIN`. `PLATFORM_ADMIN` pode operar qualquer tenant pelas mesmas rotas.

- `POST /api/v1/tenants/{tenant_id}/campaigns`
- `GET /api/v1/tenants/{tenant_id}/campaigns?status=&from=&to=&q=&limit=&offset=`
- `GET /api/v1/tenants/{tenant_id}/campaigns/{campaign_id}`
- `PUT /api/v1/tenants/{tenant_id}/campaigns/{campaign_id}` (somente `DRAFT`)
- `POST /api/v1/tenants/{tenant_id}/campaigns/{campaign_id}/imports/preview`
- `POST /api/v1/tenants/{tenant_id}/campaigns/{campaign_id}/imports/{import_id}/confirm`
- `POST /api/v1/tenants/{tenant_id}/campaigns/{campaign_id}/issuance`
- `GET /api/v1/tenants/{tenant_id}/campaigns/{campaign_id}/analytics`
- `GET /api/v1/admin/campaigns/dashboard?tenant_id=&campaign_id=&provider_id=&status=&from=&to=`

## CSV

O upload usa `Content-Type: text/csv` e este cabeçalho exato:

```csv
email,cpf_cnpj,valor,vencimento,external_id
cliente1@email.com,52998224725,1499.00,2026-09-30,CAMP-00001
cliente2@email.com,04252011000110,750.50,2026-09-30,
```

`email` é normalizado para minúsculas. `cpf_cnpj` é obrigatório, aceita pontuação, é normalizado para dígitos e tem seus verificadores validados. O documento é persistido como `payer_document`, sem criar Customer ou fabricar nome/endereço. `valor` usa ponto decimal, no máximo duas casas e é convertido para centavos sem `float`. `vencimento` usa `YYYY-MM-DD` e não pode estar no passado. `external_id` vazio vira `null` e é validado contra duplicatas do arquivo e do tenant.

Limites: 16 MiB, 100.000 registros e até 200 erros detalhados na resposta. Códigos atuais: `INVALID_EMAIL`, `INVALID_DOCUMENT`, `INVALID_AMOUNT`, `INVALID_DUE_DATE`, `DUPLICATE_EXTERNAL_ID`, `INVALID_COLUMNS` e `INVALID_FILE_TYPE`.

## Segurança e processamento

Preview não cria boletos nem chama provider. O backend faz parsing streaming, normaliza as linhas e grava uma sessão de staging vinculada ao SHA-256 do conteúdo por 24 horas. A confirmação usa essa sessão imutável e cria os boletos válidos numa transação SQL, associando o provider ativo do tenant e a campanha.

O início da emissão retorna HTTP 202 e apenas marca a campanha como `PROCESSING`. O loop periódico da aplicação reivindica lotes de até 100 boletos com `FOR UPDATE SKIP LOCKED`, reaproveita `BoletoService.Emit` e, portanto, mantém compliance, ProviderFactory, Moncalieri, idempotência, status e webhooks existentes. Claims abandonados podem ser retomados após cinco minutos. Falhas e bloqueios são isolados por boleto; a campanha termina como `COMPLETED` ou `PARTIAL`.

Analytics são agregadas em SQL e não transferem a coleção inteira de boletos ao navegador. Para grandes volumes, acompanhe o progresso pelo endpoint de analytics. Esta etapa deliberadamente não usa SQS; uma futura fila pode substituir `CampaignIssuer` sem mudar o domínio ou as rotas públicas.
