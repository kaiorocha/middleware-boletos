# Architecture

Visão geral

O sistema é composto por um backend Go, um frontend Next.js, e serviços de infraestrutura (PostgreSQL e Redis) orquestrados por Docker Compose para desenvolvimento local. A arquitetura é pensada para evoluir para múltiplos provedores bancários com isolamento multi-tenant.

Backend (Go)
- Serviço REST minimalista que expõe endpoints (ex.: /health) e serve como fachada para integrações bancárias.
- Estrutura modular: cmd/ para entrypoint, internal/ para lógica de negócio e domain models, pkg/ para utilitários reutilizáveis.
- Preparado para múltiplas instâncias em ambientes diferentes (local, dev, staging, prod).

Frontend (Next.js)
- Aplicação React/Next para interface do usuário e páginas administrativas.
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
- Abstração de Provider na camada de domínio com configurações por Tenant.
- Estratégia: adaptadores por provedor (pattern Adapter/Strategy) para isolar contratos específicos de cada banco.

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
- Não há integração bancária real nesta etapa; o foco é scaffolding, domínio e infra para desenvolvimento.
