# Roadmap

Etapas contratadas:

- Etapa 1: Fundação (concluída) — scaffolding, domain model inicial, docker-compose, frontend básico e /health endpoint.

- Etapa 2: Backend e API
  - Implementar persistência (Postgres), migrações e repositórios.
  - Criar endpoints REST para CRUD de entidades principais.
  - Validação, testes unitários e integração.

- Etapa 3: Integração Bancária
  - Desenvolver adaptadores para provedores bancários.
  - Implementar emissão de boletos simulada e rotas de homologação.
  - Testes de integração com sandboxes de bancos.

- Etapa 4: Painel Administrativo
  - Interface para gestão de Tenants, Users, Customers, Providers e Boletos.
  - Controles de acesso e auditoria (audit logs).

- Etapa 5: Infraestrutura e Deploy
  - Preparar pipelines CI/CD, infraestrutura (K8s ou serviços gerenciados), segredos e backups.
  - Monitoramento básico e alertas.

- Etapa 6: Monitoramento e Go-live
  - Testes de carga, observabilidade, runbooks e go-live controlado.

Fase 2: Plataforma Completa
- Expansão de features (conciliação automática, cobranças recorrentes, interfaces bancárias adicionais, SLA e suporte empresarial).
