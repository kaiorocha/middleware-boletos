# Architecture

Visão geral

O sistema é composto por um backend Go, um frontend Next.js, e serviços de infraestrutura (PostgreSQL e Redis) orquestrados por Docker Compose para desenvolvimento local. A arquitetura é pensada para evoluir para múltiplos provedores bancários com isolamento multi-tenant.

Backend (Go)
- Serviço REST minimalista que expõe endpoints (ex.: /health) e serve como fachada para integrações bancárias.
- Estrutura modular: cmd/ para entrypoint, internal/ para lógica de negócio e domain models, pkg/ para utilitários reutilizáveis.
- Preparado para múltiplas instâncias em ambientes diferentes (local, dev, staging, prod).

Frontend (Next.js)
- Aplicação React/Next para interface do usuário e páginas administrativas.
- Fluxo SaaS demonstrável com `/login`, `/admin` para `PLATFORM_ADMIN` e `/app` para tenants.
- Separação de build e runtime para permitir CDN/Edge deployment no futuro.

PostgreSQL
- Banco relacional primário para persistência de entidades (Tenants, Users, Customers, Boletos, AuditLogs).
- Recomendação futura: usar migrações (por exemplo, golang-migrate) e configurações por ambiente.

Redis
- Uso previsto para caching, filas leves e mecanismos de lock/coordenação.
- Implementar nomes de chaves por Tenant para isolamento.

Docker Compose
- Orquestração local para desenvolvimento com serviços: backend, frontend, postgres e redis.
- Estrutura facilita reprodução do ambiente de desenvolvimento para toda a equipe.

Preparação para múltiplos provedores
- A camada `backend/internal/providers` isola integrações bancárias por contrato.
- `providers` representa o catálogo global gerenciado pela plataforma.
- `tenant_providers` representa providers habilitados para cada tenant.
- `contracts.ProviderAdapter` define as operações comuns: emissão, consulta, listagem, cancelamento, webhook, saldo e health.
- `factory.ProviderFactory` é o único ponto de seleção de adapter por `provider.name`.
- Services consomem apenas interfaces e tipos padronizados; nenhum service conhece contratos específicos de bancos.
- `mock.Provider` é o adapter operacional desta etapa e simula emissão sem banco real.

Fluxo de emissão
- API recebe `POST /api/v1/tenants/:tenantId/boletos/:id/emit`.
- `BoletoService` valida dependências obrigatórias, tenant/boleto/provider, verifica idempotência, busca o customer, consulta a blacklist e confirma que o provider global está ativo e habilitado para o tenant antes de montar o adapter.
- O fluxo é fail-closed: sem `BlacklistService`, sem documento do customer ou com erro na blacklist, a emissão é interrompida.
- `adapter.IssueBoleto` só é chamado dentro de `BoletoService.Emit`, após autorização tenant-scoped e validação de Compliance.
- Provider não habilitado retorna `PROVIDER_NOT_ALLOWED` com HTTP 403.
- A configuração usada na factory vem de `tenant_providers.config`; `providers.config` é fallback apenas para configuração comum não sensível.
- O retorno padronizado persiste `status`, `external_id`, `barcode`, `digitable_line`, `our_number` e `issued_at`.
- Logs estruturados registram tenant, provider, request id, boleto id, latência e resultado.

Compliance
- A tabela `blacklist` armazena documentos bloqueados por tenant, com soft delete e índice único parcial para bloqueio ativo.
- `BlacklistService.IsBlocked` é obrigatório antes de qualquer emissão.
- Tentativas bloqueadas retornam `CUSTOMER_BLOCKED` com HTTP 409 e registram auditoria.

Autorização multi-tenant
- Login em `POST /api/v1/auth/login` valida senha com hash bcrypt e emite JWT HS256.
- Rotas sob `/api/v1/tenants/{tenantId}/...` passam por autorização centralizada antes dos handlers.
- A identidade autenticada vem de JWT Bearer validado e informa tenants permitidos por `tenant_id` ou `tenant_ids`.
- Sem identidade, a API retorna HTTP 401. Com identidade sem acesso ao tenant, retorna HTTP 403.
- RBAC usa a claim `roles`; `PLATFORM_ADMIN` é exigido para operações globais como listar e criar tenants.
- Usuários comuns obtêm seus tenants por `GET /api/v1/me/tenants` ou pela sessão/JWT, sem acesso à listagem global.
- Headers arbitrários como `X-User-ID`, `X-Tenant-ID` e `X-Tenant-IDs` não autenticam usuários em produção.
- Em desenvolvimento explícito (`APP_ENV=development`), `X-Dev-User-ID` e `X-Dev-Tenant-ID` permitem operação local controlada.
- CORS aceita `*` em desenvolvimento local; em produção `CORS_ALLOWED_ORIGINS` deve listar origens explícitas.

Estados de boleto
- Status suportados: `CREATED`, `PROCESSING`, `ISSUED`, `FAILED`, `CANCELLED`, `PARTIAL`, `PAID`, `EXPIRED`.
- Transições inválidas são bloqueadas pela máquina de estados em `providers/base`.

Webhooks
- `providers/webhooks.Receive` valida e converte payloads usando o adapter.
- Nesta etapa o `MockProvider` desserializa eventos padronizados e prepara a geração de eventos internos.

Estratégia de failover (futuro)
- Retry e fallback para provedores: implementar circuit breakers e filas para requisições a provedores.
- Roteamento dinâmico: se um provedor estiver indisponível, rotear para provedor alternativo configurado para o Tenant.
- Monitoramento ativo e alertas para detectar falhas e acionar automações de failover.

Ambientes
- Local: Docker Compose com dados efêmeros (volumes locais para postgres se necessário).
- Desenvolvimento: ambiente compartilhado com deploys automatizados de branches de feature.
- Homologação (staging): ambiente que replica produção para testes de integração.
- Produção: infraestrutura com orquestração (Kubernetes/managed services), backups, escalabilidade e alta disponibilidade.

Notas
- Credenciais de provider não devem ser retornadas completas por endpoints de consulta nem exibidas em texto aberto no frontend.
